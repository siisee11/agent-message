use std::fs;
use std::path::PathBuf;

fn main() {
    let manifest_dir =
        PathBuf::from(std::env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR"));
    let root_package_json = manifest_dir.join("..").join("package.json");

    println!("cargo:rerun-if-changed={}", root_package_json.display());

    let package_json = fs::read_to_string(&root_package_json).expect("read root package.json");
    let version = extract_version(&package_json).expect("extract version from root package.json");

    println!("cargo:rustc-env=APP_VERSION={version}");
}

fn extract_version(package_json: &str) -> Option<String> {
    for line in package_json.lines() {
        let trimmed = line.trim();
        if !trimmed.starts_with("\"version\"") {
            continue;
        }

        let value = trimmed
            .split_once(':')
            .map(|(_, rest)| rest.trim().trim_end_matches(','))?;
        return Some(value.trim_matches('"').to_string());
    }

    None
}
