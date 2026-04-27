use askama::Template;
use axum::{
    response::IntoResponse,
    extract::{Path, State},
    Form,
};
use chrono::Utc;

use crate::{
    models::{Log, NewLog},
    state::AppState,
    services::generate_summary,
};

// Templates
#[derive(askama::Template)]
#[template(path = "create.html")]
struct CreateTemplate;

#[derive(askama::Template)]
#[template(path = "view.html")]
struct ViewTemplate {
    log: Log,
}

// Handlers
pub async fn new_log() -> impl IntoResponse {
    let template = CreateTemplate;
    axum::response::Html(template.render().unwrap())
}

pub async fn create_log(
    State(state): State<AppState>,
    Form(new_log): Form<NewLog>,
) -> impl IntoResponse {
    let now = Utc::now();
    
    sqlx::query(
        "INSERT INTO logs (title, content, summary, created_at, updated_at) VALUES (?, ?, ?, ?, ?)"
    )
    .bind(&new_log.title)
    .bind(&new_log.content)
    .bind(None::<String>)
    .bind(now)
    .bind(now)
    .execute(&state.db)
    .await
    .expect("Failed to insert log");
    
    // Redirect to homepage
    axum::response::Redirect::to("/")
}

pub async fn view_log(
    Path(id): Path<i64>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    let log = sqlx::query_as::<_, Log>(
        "SELECT id, title, content, summary, created_at, updated_at FROM logs WHERE id = ?"
    )
    .bind(id)
    .fetch_one(&state.db)
    .await
    .expect("Failed to fetch log");
    
    let template = ViewTemplate { log };
    axum::response::Html(template.render().unwrap())
}

pub async fn delete_log(
    Path(id): Path<i64>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    sqlx::query("DELETE FROM logs WHERE id = ?")
        .bind(id)
        .execute(&state.db)
        .await
        .expect("Failed to delete log");
        
    // Redirect to homepage
    axum::response::Redirect::to("/")
}

pub async fn summarize_log(
    Path(id): Path<i64>,
    State(state): State<AppState>,
) -> impl IntoResponse {
    let log = sqlx::query_as::<_, Log>(
        "SELECT id, title, content, summary, created_at, updated_at FROM logs WHERE id = ?"
    )
    .bind(id)
    .fetch_one(&state.db)
    .await
    .expect("Failed to fetch log");
    
    let summary = generate_summary(&log.content)
        .await
        .unwrap_or_else(|_| "Failed to generate summary".to_string());
    
    sqlx::query("UPDATE logs SET summary = ?, updated_at = ? WHERE id = ?")
        .bind(&summary)
        .bind(Utc::now())
        .bind(id)
        .execute(&state.db)
    .await
    .expect("Failed to update log summary");
    
    // Redirect to log view
    axum::response::Redirect::to(&format!("/logs/{}", id))
}