use axum::{response::IntoResponse, extract::State};
use askama::Template;

use crate::{models::LogSummary, state::AppState};

// Define template at the top level
#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    logs: Vec<LogSummary>,
}

pub async fn home(State(state): State<AppState>) -> impl IntoResponse {
    // Fetch logs from database
    let logs = sqlx::query_as::<_, LogSummary>(
        "SELECT id, title, summary, created_at FROM logs ORDER BY created_at DESC"
    )
    .fetch_all(&state.db)
    .await
    .unwrap_or_else(|_| vec![]);
    
    let template = IndexTemplate { logs };
    axum::response::Html(template.render().unwrap())
}