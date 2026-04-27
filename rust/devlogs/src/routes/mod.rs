pub mod home;
pub mod logs;

use axum::{
    routing::{get, post},
    Router,
};
use sqlx::SqlitePool;

use crate::{routes::home::*, routes::logs::*};
use crate::state::AppState;

pub fn create_router(pool: SqlitePool) -> Router {
    let app_state = AppState { db: pool };
    
    Router::new()
        .route("/", get(home))
        .route("/logs/new", get(new_log))
        .route("/logs", post(create_log))
        .route("/logs/:id", get(view_log))
        .route("/logs/:id/delete", post(delete_log))
        .route("/logs/:id/summarize", post(summarize_log))
        .with_state(app_state)
}