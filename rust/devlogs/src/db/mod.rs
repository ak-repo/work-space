pub mod pool;

use sqlx::SqlitePool;

pub async fn setup_database() -> Result<SqlitePool, sqlx::Error> {
    let pool = SqlitePool::connect("sqlite:devlogs.db").await?;
    Ok(pool)
}