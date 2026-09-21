//! ChaosMeshLite: chaos experiments as CI-runnable SLO assertions.
//!
//! Inject deterministic faults (added latency, periodic errors) into a stream
//! of observed call latencies, then assert that Service Level Objectives still
//! hold. The fault schedule is deterministic (not RNG) so results are
//! reproducible and identical across every language port.

pub mod chaos;

pub use chaos::{
    evaluate_slo, percentile, run_experiment, ExperimentResult, FaultPolicy, Slo, SloResult,
};
