#![feature(once_cell)]

pub mod checker {
    use crate::utils::{c_char_to_vec, vec_to_c_char};
    use libc::c_char;
    use prover::zkevm::{CircuitCapacityChecker, RowUsage};
    use serde_derive::{Deserialize, Serialize};
    use std::cell::OnceCell;
    use std::collections::HashMap;
    use std::panic;
    use std::ptr::null;
    use types::eth::BlockTrace;

    #[derive(Debug, Clone, Deserialize, Serialize)]
    pub struct RowUsageResult {
        pub acc_row_usage: Option<RowUsage>,
        pub tx_row_usage: Option<RowUsage>,
        pub error: Option<String>,
    }

    static mut CHECKERS: OnceCell<HashMap<u64, CircuitCapacityChecker>> = OnceCell::new();

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn init() {
        env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("debug"))
            .format_timestamp_millis()
            .init();
        let checkers = HashMap::new();
        CHECKERS
            .set(checkers)
            .expect("circuit capacity checker initialized twice");
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn new_circuit_capacity_checker() -> u64 {
        let checkers = CHECKERS
            .get_mut()
            .expect("fail to get circuit capacity checkers map in new_circuit_capacity_checker");
        let id = checkers.len() as u64;
        let checker = CircuitCapacityChecker::new();
        checkers.insert(id, checker);
        id
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn reset_circuit_capacity_checker(id: u64) {
        CHECKERS
            .get_mut()
            .expect("fail to get circuit capacity checkers map in reset_circuit_capacity_checker")
            .get_mut(&id)
            .unwrap_or_else(|| panic!("fail to get circuit capacity checker (id: {id:?}) in reset_circuit_capacity_checker"))
            .reset()
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn apply_tx(id: u64, tx_traces: *const c_char) -> *const c_char {
        let result = panic::catch_unwind(|| {
            let tx_traces_vec = c_char_to_vec(tx_traces);
            let traces = serde_json::from_slice::<BlockTrace>(&tx_traces_vec)
                .unwrap_or_else(|_| panic!("id: {id:?}, fail to deserialize tx_traces"));
            if traces.transactions.len() != 1 {
                panic!("traces.transactions.len() != 1".to_string())
            } else if traces.execution_results.len() != 1 {
                panic!("traces.execution_results.len() != 1".to_string())
            } else if traces.tx_storage_trace.len() != 1 {
                panic!("traces.tx_storage_trace.len() != 1".to_string())
            }
            CHECKERS
                .get_mut()
                .expect("fail to get circuit capacity checkers map in apply_tx")
                .get_mut(&id)
                .unwrap_or_else(|| {
                    panic!(
                        "fail to get circuit capacity checker (id: {id:?}) in apply_tx"
                    )
                })
                .estimate_circuit_capacity(&[traces.clone()])
                .unwrap_or_else(|_| {
                    panic!(
                        "id: {:?}, fail to estimate_circuit_capacity in apply_tx, block_hash: {:?}, tx_hash: {:?}",
                        id, traces.header.hash, traces.transactions[0].tx_hash
                    )
                })
        });
        let r = match result {
            Ok((acc_row_usage, tx_row_usage)) => {
                log::debug!(
                    "id: {:?}, acc_row_usage: {:?}, tx_row_usage: {:?}",
                    id,
                    acc_row_usage.row_number,
                    tx_row_usage.row_number
                );
                RowUsageResult {
                    acc_row_usage: Some(acc_row_usage),
                    tx_row_usage: Some(tx_row_usage),
                    error: None,
                }
            }
            Err(e) => RowUsageResult {
                acc_row_usage: None,
                tx_row_usage: None,
                error: Some(format!("{e:?}")),
            },
        };
        serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn apply_block(id: u64, block_trace: *const c_char) -> *const c_char {
        let result = panic::catch_unwind(|| {
            let block_trace = c_char_to_vec(block_trace);
            let traces = serde_json::from_slice::<BlockTrace>(&block_trace)
                .unwrap_or_else(|_| panic!("id: {id:?}, fail to deserialize block_trace"));
            CHECKERS
                .get_mut()
                .expect("fail to get circuit capacity checkers map in apply_block")
                .get_mut(&id)
                .unwrap_or_else(|| {
                    panic!(
                        "fail to get circuit capacity checker (id: {id:?}) in apply_block"
                    )
                })
                .estimate_circuit_capacity(&[traces.clone()])
                .unwrap_or_else(|_| {
                    panic!(
                        "id: {:?}, fail to estimate_circuit_capacity in apply_block, block_hash: {:?}",
                        id, traces.header.hash
                    )
                })
        });
        let r = match result {
            Ok((acc_row_usage, tx_row_usage)) => {
                log::debug!(
                    "id: {:?}, acc_row_usage: {:?}, tx_row_usage: {:?}",
                    id,
                    acc_row_usage.row_number,
                    tx_row_usage.row_number
                );
                RowUsageResult {
                    acc_row_usage: Some(acc_row_usage),
                    tx_row_usage: Some(tx_row_usage),
                    error: None,
                }
            }
            Err(e) => RowUsageResult {
                acc_row_usage: None,
                tx_row_usage: None,
                error: Some(format!("{e:?}")),
            },
        };
        serde_json::to_vec(&r).map_or(null(), vec_to_c_char)
    }
}

pub(crate) mod utils {
    use std::ffi::{CStr, CString};
    use std::os::raw::c_char;

    #[allow(dead_code)]
    pub(crate) fn c_char_to_str(c: *const c_char) -> &'static str {
        let cstr = unsafe { CStr::from_ptr(c) };
        cstr.to_str().expect("fail to cast cstr to str")
    }

    #[allow(dead_code)]
    pub(crate) fn c_char_to_vec(c: *const c_char) -> Vec<u8> {
        let cstr = unsafe { CStr::from_ptr(c) };
        cstr.to_bytes().to_vec()
    }

    #[allow(dead_code)]
    pub(crate) fn vec_to_c_char(bytes: Vec<u8>) -> *const c_char {
        CString::new(bytes)
            .expect("fail to create new CString from bytes")
            .into_raw()
    }

    #[allow(dead_code)]
    pub(crate) fn bool_to_int(b: bool) -> u8 {
        match b {
            true => 1,
            false => 0,
        }
    }
}
