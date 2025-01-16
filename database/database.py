import sqlite3
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, List, Optional

class Database:
    def __init__(self, db_path: str = "data/database.sqlite"):
        self.db_path = db_path
        self.conn = None
        self.cursor = None
        
    def connect(self):
        """Establish database connection"""
        try:
            # Ensure directory exists
            Path(self.db_path).parent.mkdir(parents=True, exist_ok=True)
            
            self.conn = sqlite3.connect(self.db_path)
            self.conn.row_factory = sqlite3.Row  # Enable row factory for named columns
            self.cursor = self.conn.cursor()
            logging.info(f"Connected to database: {self.db_path}")
        except sqlite3.Error as e:
            logging.error(f"Error connecting to database: {e}")
            raise

    def close(self):
        """Close database connection"""
        if self.conn:
            self.conn.close()
            logging.info("Database connection closed")

    def init_database(self):
        """Initialize database schema"""
        try:
            self.cursor.executescript("""
                CREATE TABLE IF NOT EXISTS announcements (
                    id INTEGER PRIMARY KEY,
                    title TEXT NOT NULL,
                    link TEXT UNIQUE NOT NULL,
                    published_date DATE,
                    description TEXT,
                    project_id TEXT,
                    dept_id TEXT,
                    announce_type TEXT,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                );

                CREATE TABLE IF NOT EXISTS downloads (
                    id INTEGER PRIMARY KEY,
                    announcement_id INTEGER,
                    file_path TEXT,
                    download_status TEXT,
                    download_date TIMESTAMP,
                    FOREIGN KEY (announcement_id) REFERENCES announcements(id)
                );

                CREATE TABLE IF NOT EXISTS procurement_details (
                    id INTEGER PRIMARY KEY,
                    announcement_id INTEGER,
                    budget_amount DECIMAL,
                    quantity INTEGER,
                    duration_years INTEGER,
                    duration_months INTEGER,
                    submission_date DATE,
                    submission_time TIME,
                    contact_phone TEXT,
                    contact_email TEXT,
                    extracted_at TIMESTAMP,
                    FOREIGN KEY (announcement_id) REFERENCES announcements(id)
                );

                -- Create indexes for better query performance
                CREATE INDEX IF NOT EXISTS idx_announcements_link ON announcements(link);
                CREATE INDEX IF NOT EXISTS idx_downloads_announcement_id ON downloads(announcement_id);
                CREATE INDEX IF NOT EXISTS idx_procurement_announcement_id ON procurement_details(announcement_id);
            """)
            self.conn.commit()
            logging.info("Database schema initialized successfully")
        except sqlite3.Error as e:
            logging.error(f"Error initializing database schema: {e}")
            raise

    def insert_announcement(self, announcement: Dict[str, Any], dept_id: Optional[str] = None) -> Optional[int]:
        """
        Insert a new announcement into the database
        Args:
            announcement: Announcement data dictionary
            dept_id: Department ID that was used in the feed request
        Returns the ID of the inserted row
        """
        try:
            # Extract project_id and announcement type from description
            description = announcement.get('description', '')
            project_id = None
            announce_type = None
            
            # Description format: "67119457432, ประกวดราคาอิเล็กทรอนิกส์ (e-bidding), ประกาศเชิญชวน"
            if description:
                parts = description.split(',')
                if parts:
                    project_id = parts[0].strip()
                    if len(parts) > 2:
                        announce_type = parts[2].strip()

            self.cursor.execute("""
                INSERT OR REPLACE INTO announcements (
                    title, link, published_date, description,
                    project_id, dept_id, announce_type,
                    updated_at
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
            """, (
                announcement['title'],
                announcement['link'],
                announcement['published_date'],
                description,
                project_id,
                dept_id,  # Use the department ID from the request
                announce_type
            ))
            self.conn.commit()
            return self.cursor.lastrowid
        except sqlite3.Error as e:
            logging.error(f"Error inserting announcement: {e}")
            return None

    def insert_download(self, announcement_id: int, file_path: str, status: str) -> Optional[int]:
        """Insert a new download record"""
        try:
            self.cursor.execute("""
                INSERT INTO downloads (announcement_id, file_path, download_status, download_date)
                VALUES (?, ?, ?, CURRENT_TIMESTAMP)
            """, (announcement_id, file_path, status))
            self.conn.commit()
            return self.cursor.lastrowid
        except sqlite3.Error as e:
            logging.error(f"Error inserting download: {e}")
            return None

    def get_pending_downloads(self) -> List[Dict[str, Any]]:
        """Get announcements that haven't been downloaded yet"""
        try:
            self.cursor.execute("""
                SELECT a.id, a.link
                FROM announcements a
                LEFT JOIN downloads d ON a.id = d.announcement_id
                WHERE d.id IS NULL
            """)
            return [dict(row) for row in self.cursor.fetchall()]
        except sqlite3.Error as e:
            logging.error(f"Error getting pending downloads: {e}")
            return []

    def get_recent_announcements(self, dept_id: Optional[str] = None, limit: int = 10) -> List[Dict]:
        """Get recent announcements with optional department filter"""
        try:
            # Print debug info
            debug_query = """SELECT COUNT(*) FROM announcements"""
            self.cursor.execute(debug_query)
            total_count = self.cursor.fetchone()[0]
            logging.info(f"Total announcements in database: {total_count}")

            if dept_id:
                query = """
                    SELECT id, title, link, published_date, description, 
                           created_at, project_id, dept_id, announce_type, updated_at
                    FROM announcements 
                    WHERE dept_id = ?
                    ORDER BY updated_at DESC
                    LIMIT ?
                """
                self.cursor.execute(query, (dept_id, limit))

                # Debug logging
                logging.info(f"Running query with dept_id: {dept_id}")
                logging.info("Sample of stored data:")
                debug_query = "SELECT title, description FROM announcements LIMIT 1"
                self.cursor.execute(debug_query)
                sample = self.cursor.fetchone()
                if sample:
                    logging.info(f"Sample title: {sample[0][:100]}")
                    logging.info(f"Sample description: {sample[1][:100] if sample[1] else 'None'}")
            else:
                query = """
                    SELECT id, title, link, published_date, description, created_at 
                    FROM announcements 
                    ORDER BY created_at DESC
                    LIMIT ?
                """
                self.cursor.execute(query, (limit,))
                
            results = [dict(row) for row in self.cursor.fetchall()]
            logging.info(f"Found {len(results)} announcements matching criteria")
            return results
        except sqlite3.Error as e:
            logging.error(f"Error getting recent announcements: {e}")
            raise

    def update_download_status(self, announcement_id: int, status: str):
        """Update the download status for an announcement"""
        try:
            self.cursor.execute("""
                UPDATE downloads
                SET download_status = ?, download_date = CURRENT_TIMESTAMP
                WHERE announcement_id = ?
            """, (status, announcement_id))
            self.conn.commit()
        except sqlite3.Error as e:
            logging.error(f"Error updating download status: {e}")

    def __enter__(self):
        """Context manager enter"""
        self.connect()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit"""
        self.close()