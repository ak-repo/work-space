use sqlx::sqlite::SqlitePoolOptions;
use sqlx::SqlitePool;

pub async fn connect_to_database(database_url: &str) -> Result<SqlitePool, sqlx::Error> {
    let pool = SqlitePoolOptions::new()
        .max_connections(5)
        .connect(database_url)
        .await?;

    Ok(pool)
}