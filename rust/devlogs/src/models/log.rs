use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::FromRow;

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct Log {
    pub id: i64,
    pub title: String,
    pub content: String,
    pub summary: Option<String>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NewLog {
    pub title: String,
    pub content: String,
}

#[derive(Debug, Clone, FromRow, Serialize, Deserialize)]
pub struct LogSummary {
    pub id: i64,
    pub title: String,
    pub summary: Option<String>,
    pub created_at: DateTime<Utc>,
}