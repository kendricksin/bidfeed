import sqlite3
import csv
import sys
from pathlib import Path
import logging
from datetime import datetime

def setup_logging():
    """Configure logging"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(),
            logging.FileHandler('data/export_db.log')
        ]
    )

def export_table_to_csv(conn, table_name, output_dir):
    """Export a single table to CSV"""
    cursor = conn.cursor()
    
    # Get column names
    cursor.execute(f"PRAGMA table_info({table_name})")
    columns = [col[1] for col in cursor.fetchall()]
    
    # Get all data from the table
    cursor.execute(f"SELECT * FROM {table_name}")
    rows = cursor.fetchall()
    
    # Generate filename with timestamp
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    output_file = output_dir / f"{table_name}_{timestamp}.csv"
    
    # Write to CSV
    with open(output_file, 'w', newline='', encoding='utf-8') as f:
        writer = csv.writer(f)
        writer.writerow(columns)  # Write header
        writer.writerows(rows)    # Write data
        
    logging.info(f"Exported {len(rows)} rows from {table_name} to {output_file}")
    return len(rows)

def main():
    """Main function to export all tables to CSV"""
    setup_logging()
    logging.info("Starting database export...")
    
    try:
        # Setup paths
        db_path = Path("data/database.sqlite")
        output_dir = Path("data/exports")
        output_dir.mkdir(parents=True, exist_ok=True)
        
        # Connect to database
        conn = sqlite3.connect(db_path)
        conn.row_factory = sqlite3.Row
        
        # Get list of tables
        cursor = conn.cursor()
        cursor.execute("SELECT name FROM sqlite_master WHERE type='table'")
        tables = [row[0] for row in cursor.fetchall()]
        
        total_rows = 0
        for table in tables:
            rows_exported = export_table_to_csv(conn, table, output_dir)
            total_rows += rows_exported
            
        logging.info(f"Export completed successfully. Total rows exported: {total_rows}")
        
        # Close connection
        conn.close()
        
    except sqlite3.Error as e:
        logging.error(f"Database error: {e}")
        sys.exit(1)
    except Exception as e:
        logging.error(f"Error during export: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()