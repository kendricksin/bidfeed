import logging
import sys
from pathlib import Path
import sqlite3

# Add parent directory to Python path
sys.path.append(str(Path(__file__).parent.parent))

def setup_logging():
    """Configure logging"""
    log_dir = Path("data/logs")
    log_dir.mkdir(parents=True, exist_ok=True)
    
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(),
            logging.FileHandler('data/init_db.log')
        ]
    )

def init_database(db_path: str = "data/database.sqlite"):
    """Initialize database with new schema"""
    try:
        # Ensure directory exists
        Path(db_path).parent.mkdir(parents=True, exist_ok=True)
        
        # Connect to database
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()
        
        # Create tables with new schema
        cursor.executescript("""
            -- Drop existing tables if they exist
            DROP TABLE IF EXISTS procurement_details;
            DROP TABLE IF EXISTS downloads;
            DROP TABLE IF EXISTS announcements;
            
            -- Create announcements table with new schema
            CREATE TABLE announcements (
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
            
            -- Create downloads table
            CREATE TABLE downloads (
                id INTEGER PRIMARY KEY,
                announcement_id INTEGER,
                file_path TEXT,
                download_status TEXT,
                download_date TIMESTAMP,
                FOREIGN KEY (announcement_id) REFERENCES announcements(id)
            );
            
            -- Create procurement details table
            CREATE TABLE procurement_details (
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
            CREATE INDEX idx_announcements_link ON announcements(link);
            CREATE INDEX idx_announcements_dept_id ON announcements(dept_id);
            CREATE INDEX idx_announcements_project_id ON announcements(project_id);
            CREATE INDEX idx_announcements_updated ON announcements(updated_at);
            CREATE INDEX idx_downloads_announcement_id ON downloads(announcement_id);
            CREATE INDEX idx_procurement_announcement_id ON procurement_details(announcement_id);
        """)
        
        conn.commit()
        logging.info("Database schema initialized successfully")
        
        # Close connection
        conn.close()
        
    except sqlite3.Error as e:
        logging.error(f"Error initializing database: {e}")
        raise

def main():
    """Initialize the database"""
    setup_logging()
    logging.info("Starting database initialization...")
    
    try:
        init_database()
        logging.info("Database initialization completed successfully")
    except Exception as e:
        logging.error(f"Database initialization failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()