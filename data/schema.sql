-- Intermediate storage
CREATE TABLE feed_entries (
    id TEXT PRIMARY KEY,
    dept_id TEXT,
    title TEXT,
    pdf_url TEXT,
    publish_date TEXT,
    status TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- Main storage
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    feed_entry_id TEXT,
    title TEXT,
    dept_id TEXT,
    budget REAL,
    pdf_content TEXT,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(feed_entry_id) REFERENCES feed_entries(id)
);

-- Error tracking
CREATE TABLE errors (
    id TEXT PRIMARY KEY,
    source TEXT,
    error_type TEXT,
    message TEXT,
    retry_count INTEGER,
    created_at TEXT DEFAULT CURRENT_TIMESTAMP
);