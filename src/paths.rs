use crate::{Error, Result};
use std::path::PathBuf;

/// Return the standard replay path for this OS without checking its existence.
pub fn default_replay_dir() -> Result<PathBuf> {
    #[cfg(target_os = "macos")]
    {
        let home = std::env::var_os("HOME")
            .ok_or_else(|| Error::new("REPLAY_FOLDER_NOT_FOUND", "HOME is not set; pass --dir"))?;
        Ok(PathBuf::from(home)
            .join("Library/Containers/com.tanksblitz/Data/Documents/tb-project/replays"))
    }
    #[cfg(target_os = "windows")]
    {
        let profile = std::env::var_os("USERPROFILE").ok_or_else(|| {
            Error::new(
                "REPLAY_FOLDER_NOT_FOUND",
                "USERPROFILE is not set; pass --dir",
            )
        })?;
        Ok(PathBuf::from(profile).join("Documents/TanksBlitz/replays"))
    }
    #[cfg(not(any(target_os = "macos", target_os = "windows")))]
    {
        Err(Error::new(
            "REPLAY_FOLDER_NOT_FOUND",
            "no standard replay path is configured for this OS; pass --dir",
        ))
    }
}
