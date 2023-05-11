#![feature(once_cell)]

pub mod checker {
    use crate::utils::{c_char_to_str, c_char_to_vec, vec_to_c_char};
    use libc::c_char;
    use std::cell::OnceCell;
    use std::panic;
    use std::ptr::null;
    use types::eth::BlockTrace;
    use zkevm::circuit::AGG_DEGREE;
    use zkevm::{capacity_checker::CircuitCapacityChecker};

    static mut CHECKER: OnceCell<CircuitCapacityChecker> = OnceCell::new();

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn new_circuit_capacity_checker() {
        // TODO: better logger
        env_logger::init();

        let c = CircuitCapacityChecker::new();
        CHECKER.set(c).unwrap();
    }

    /// # Safety
    #[no_mangle]
    pub unsafe extern "C" fn reset_circuit_capacity_checker() {
        CHECKER.get_mut().unwrap().reset()
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
}
