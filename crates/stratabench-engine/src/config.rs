use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Deserialize)]
pub struct EngineConfig {
    pub target: String,
    #[serde(default)]
    pub profile: String,
    #[serde(default)]
    pub layer: String,
    #[serde(default)]
    pub pattern: String,
    #[serde(default = "default_block_size")]
    pub block_size: String,
    #[serde(default)]
    pub dataset_size: String,
    #[serde(default = "default_duration")]
    pub duration_sec: i32,
    #[serde(default)]
    pub ramp_sec: i32,
    #[serde(default = "default_qd")]
    pub queue_depth: i32,
    #[serde(default = "default_threads")]
    pub threads: i32,
    #[serde(default)]
    pub read_write_mix: i32,
    #[serde(default)]
    pub direct_io: bool,
    #[serde(default)]
    pub params: HashMap<String, serde_json::Value>,
}

fn default_block_size() -> String {
    "4k".into()
}
fn default_duration() -> i32 {
    60
}
fn default_qd() -> i32 {
    32
}
fn default_threads() -> i32 {
    4
}

#[derive(Debug, Serialize)]
pub struct LatencyUS {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub min: Option<f64>,
    pub mean: Option<f64>,
    pub p50: f64,
    pub p95: f64,
    pub p99: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max: Option<f64>,
}

#[derive(Debug, Serialize)]
pub struct IntervalSample {
    pub seq: i32,
    pub elapsed_sec: f64,
    pub iops: f64,
    pub throughput_mbps: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avg_latency_us: Option<f64>,
}

#[derive(Debug, Serialize)]
pub struct EngineResults {
    pub iops: f64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub read_iops: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub write_iops: Option<f64>,
    pub throughput_mbps: f64,
    pub ops_per_sec: f64,
    pub latency_us: LatencyUS,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub total_operations: Option<i64>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub intervals: Vec<IntervalSample>,
}

pub fn parse_block_bytes(s: &str) -> u64 {
    let lower = s.trim().to_lowercase();
    let (num_str, mult): (&str, u64) = if lower.ends_with("kib") || lower.ends_with("kb") {
        (lower.trim_end_matches("kib").trim_end_matches("kb"), 1024)
    } else if lower.ends_with("mib") || lower.ends_with("mb") {
        (lower.trim_end_matches("mib").trim_end_matches("mb"), 1024 * 1024)
    } else if lower.ends_with("gib") || lower.ends_with("gb") {
        (lower.trim_end_matches("gib").trim_end_matches("gb"), 1024 * 1024 * 1024)
    } else {
        (lower.as_str(), 1)
    };
    let n: f64 = num_str.parse().unwrap_or(4096.0);
    ((n * mult as f64) as u64).max(512)
}

pub fn parse_size_bytes(s: &str) -> u64 {
    if s.is_empty() {
        return 512 * 1024 * 1024;
    }
    parse_block_bytes(s)
}
