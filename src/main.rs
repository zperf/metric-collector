use crate::Commands::Run;
use anyhow::Result;
use clap::{Args, Parser, Subcommand};
use reqwest::StatusCode;
use std::iter::zip;
use std::time::Duration;
use tokio::select;
use tokio::task::JoinSet;
use tokio_util::sync::CancellationToken;
use tracing::error;

#[derive(Parser)]
#[command(arg_required_else_help = true)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Args, Clone)]
struct RunConfig {
    #[arg(long, short, default_value = "10s")]
    collect_interval: humantime::Duration,

    #[arg(long, default_value = "9s")]
    collect_timeout: humantime::Duration,

    #[arg(long, short, default_value = "10s")]
    push_timeout: humantime::Duration,

    #[arg(long, short, default_value_t = hostname::get().unwrap().into_string().unwrap())]
    instance: String,

    #[arg(long, short, required = true)]
    jobs: Vec<String>,

    #[arg(long, short, required = true)]
    sources: Vec<String>,

    #[arg(long, short, required = true)]
    targets: Vec<String>,
}

#[derive(Subcommand)]
enum Commands {
    Run {
        #[clap(flatten)]
        config: RunConfig,
    },
}

async fn fetch_exporter(url: &String, timeout: Duration) -> Result<String> {
    let fetch = async move {
        let rsp = reqwest::get(url).await?;
        let body = rsp.text().await?;
        Result::<String>::Ok(body)
    };
    let body = tokio::time::timeout(timeout, fetch).await??;
    Ok(body)
}

async fn worker(config: RunConfig) {
    let RunConfig {
        sources,
        jobs,
        collect_timeout,
        targets,
        instance,
        push_timeout,
        ..
    } = config;
    let mut tasks: JoinSet<Result<()>> = JoinSet::new();
    for (source, job) in zip(sources, jobs) {
        let collect_timeout = *collect_timeout;
        let source = source.clone();
        let targets = targets.clone();
        let instance = instance.clone();
        tasks.spawn(async move {
            let metrics = fetch_exporter(&source, collect_timeout).await?;
            for target in targets.iter() {
                let push_url = format!("{}/job/{}/instance/{}", target, job, instance);
                let metrics = metrics.clone();
                let push_task = async move {
                    let client = reqwest::Client::new();
                    client.post(push_url).body(metrics).send().await
                };
                let rsp = tokio::time::timeout(*push_timeout, push_task).await??;
                let code = rsp.status();
                if code != StatusCode::OK {
                    let body = rsp.text().await?;
                    error!("push failed, status code: {}, body: {}", code, body);
                }
            }
            Ok(())
        });
    }
    tasks.join_all().await;
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();

    let cli = Cli::parse();
    match cli.command {
        Run { config } => {
            let mut interval = tokio::time::interval(*config.collect_interval);
            let ctrl_c = tokio::signal::ctrl_c();
            let cancel = CancellationToken::new();
            let background_task = {
                let cancel = cancel.clone();
                tokio::spawn(async move {
                    loop {
                        select! {
                            _ = cancel.cancelled() => {
                                break;
                            }

                            _ = interval.tick() => {
                                worker(config.clone()).await;
                            }
                        }
                    }
                })
            };

            ctrl_c.await?;
            cancel.cancel();
            background_task.await?;
        }
    }
    Ok(())
}
