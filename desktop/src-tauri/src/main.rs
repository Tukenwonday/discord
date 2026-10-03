// Keep the Windows console hidden in release builds while still showing panics
// and logs during development.
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    cordis_lib::run()
}