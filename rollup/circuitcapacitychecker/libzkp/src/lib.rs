#![feature(once_cell)]

pub mod checker {
    use crate::utils::{bool_to_int, c_char_to_vec};
    use libc::c_char;
    use std::cell::OnceCell;
    use std::collections::HashMap;
    use std::panic;
    use types::eth::BlockTrace;
    use zkevm::capacity_checker::CircuitCapacityChecker;

    static mut CHECKERS: OnceCell<HashMap<u64, CircuitCapacityChecker>> = OnceCell::new();

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn init() {
        env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("debug"))
            .format_timestamp_millis()
            .init();
        let checkers = HashMap::new();
        CHECKERS.set(checkers).unwrap();
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn new_circuit_capacity_checker() -> u64 {
        let id = CHECKERS.get_mut().unwrap().len() as u64;
        let checker = CircuitCapacityChecker::new();
        CHECKERS.get_mut().unwrap().insert(id, checker);
        id
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn reset_circuit_capacity_checker(id: u64) {
        CHECKERS.get_mut().unwrap().get_mut(&id).unwrap().reset()
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn apply_tx(id: u64, tx_traces: *const c_char) -> c_char {
        let tx_traces_vec = c_char_to_vec(tx_traces);
        let traces = serde_json::from_slice::<BlockTrace>(&tx_traces_vec).unwrap();
        let result = panic::catch_unwind(|| {
            CHECKERS
                .get_mut()
                .unwrap()
                .get_mut(&id)
                .unwrap()
                .estimate_circuit_capacity(&[traces])
                .unwrap()
        });
        match result {
            Ok((acc_row_usage, tx_row_usage)) => {
                if acc_row_usage.is_ok {
                    return 0 as c_char; // row usage ok
                } else if tx_row_usage.is_ok {
                    return 1 as c_char; // block row usage overflow, but tx row usage ok
                } else {
                    return 2 as c_char; // tx row usage overflow
                }
            }
            Err(_) => return 3 as c_char, // other errors than circuit capacity overflow
        }
    }
}

pub(crate) mod utils {
    use std::ffi::{CStr, CString};
    use std::os::raw::c_char;

    pub(crate) fn c_char_to_str(c: *const c_char) -> &'static str {
        let cstr = unsafe { CStr::from_ptr(c) };
        cstr.to_str().unwrap()
    }

    pub(crate) fn c_char_to_vec(c: *const c_char) -> Vec<u8> {
        let cstr = unsafe { CStr::from_ptr(c) };
        cstr.to_bytes().to_vec()
    }

    pub(crate) fn vec_to_c_char(bytes: Vec<u8>) -> *const c_char {
        CString::new(bytes).unwrap().into_raw()
    }

    pub(crate) fn bool_to_int(b: bool) -> u8 {
        match b {
            true => 1,
            false => 0,
        }
    }
}
