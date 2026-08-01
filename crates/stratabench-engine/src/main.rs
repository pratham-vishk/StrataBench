//! Native StrataBench engine — Rust block I/O (Linux O_DIRECT) with synthetic fallback.

mod block;
mod config;
mod progress;
mod synthetic;

use clap::{Parser, Subcommand};
use config::EngineConfig;
use std::fs;
use std::path::PathBuf;

const VERSION: &str = "0.2.0-io_uring";

#[derive(Parser)]
#[command(name = "stratabench-engine", version = VERSION)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    Run {
        #[arg(long)]
        config: PathBuf,
        #[arg(long)]
        output: PathBuf,
    },
    Version,
}

fn run_benchmark(cfg: &EngineConfig) -> config::EngineResults {
    if cfg.layer == "block" && !cfg.target.is_empty() {
        match block::run_block(cfg) {
            Ok(res) => return res,
            Err(e) => eprintln!("stratabench-engine: block I/O unavailable ({e}), using synthetic"),
        }
    }
    synthetic::synthesize_with_progress(cfg)
}

fn main() {
    let cli = Cli::parse();
    match cli.command {
        Commands::Version => {
            #[cfg(target_os = "linux")]
            println!("stratabench-engine {VERSION} (rust, linux block I/O)");
            #[cfg(not(target_os = "linux"))]
            println!("stratabench-engine {VERSION} (rust, synthetic only)");
        }
        Commands::Run { config, output } => {
            let raw = fs::read_to_string(&config).expect("read config");
            let cfg: EngineConfig = serde_json::from_str(&raw).expect("parse config");
            let res = run_benchmark(&cfg);
            let json = serde_json::to_string_pretty(&res).expect("serialize");
            fs::write(&output, json).expect("write output");
        }
    }
}
