//! Native StrataBench engine — Rust scaffold implementing the Go reference contract.
//! Swap `bin/stratabench-engine` (Go stub) with this binary via STRATABENCH_ENGINE_BIN.

use clap::{Parser, Subcommand};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "stratabench-engine", version = "0.1.0-stub")]
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

#[derive(Debug, Deserialize)]
struct EngineConfig {
    target: String,
    profile: String,
    layer: String,
    pattern: String,
    block_size: String,
    duration_sec: i32,
    queue_depth: i32,
    threads: i32,
    read_write_mix: i32,
}

#[derive(Debug, Serialize)]
struct LatencyUS {
    p50: f64,
    p95: f64,
    p99: f64,
}

#[derive(Debug, Serialize)]
struct EngineResults {
    iops: f64,
    read_iops: f64,
    write_iops: f64,
    throughput_mbps: f64,
    ops_per_sec: f64,
    total_operations: i64,
    latency_us: LatencyUS,
}

fn block_bytes(s: &str) -> f64 {
    let lower = s.to_lowercase();
    let (num, mult): (f64, f64) = if lower.ends_with("kib") || lower.ends_with("kb") {
        (lower.trim_end_matches("kib").trim_end_matches("kb").parse().unwrap_or(4.0), 1024.0)
    } else if lower.ends_with("mib") || lower.ends_with("mb") {
        (lower.trim_end_matches("mib").trim_end_matches("mb").parse().unwrap_or(1.0), 1024.0 * 1024.0)
    } else {
        (lower.parse().unwrap_or(4096.0), 1.0)
    };
    num * mult
}

fn synthesize(cfg: &EngineConfig) -> EngineResults {
    let threads = cfg.threads.max(1) as f64;
    let qd = cfg.queue_depth.max(1) as f64;
    let duration = cfg.duration_sec.max(1) as f64;
    let mut base = 40_000.0 * threads * qd.sqrt();
    if cfg.layer == "object" {
        base = 4_000.0 * threads;
    }
    if cfg.pattern.to_lowercase().contains("seq") {
        base *= 1.8;
    }
    let read = if cfg.read_write_mix > 0 {
        base * cfg.read_write_mix as f64 / 100.0
    } else {
        base * 0.7
    };
    let write = base - read;
    let p50 = 80.0 + qd * 2.0;
    EngineResults {
        iops: base,
        read_iops: read,
        write_iops: write,
        throughput_mbps: base * block_bytes(&cfg.block_size) / (1024.0 * 1024.0),
        ops_per_sec: base,
        total_operations: (base * duration) as i64,
        latency_us: LatencyUS {
            p50,
            p95: p50 * 2.5,
            p99: p50 * 4.0,
        },
    }
}

fn main() {
    let cli = Cli::parse();
    match cli.command {
        Commands::Version => println!("stratabench-engine 0.1.0-stub (rust scaffold)"),
        Commands::Run { config, output } => {
            let raw = fs::read_to_string(&config).expect("read config");
            let cfg: EngineConfig = serde_json::from_str(&raw).expect("parse config");
            let res = synthesize(&cfg);
            let json = serde_json::to_string_pretty(&res).expect("serialize");
            fs::write(&output, json).expect("write output");
        }
    }
}
