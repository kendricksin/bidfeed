import logging
import sys
from pathlib import Path
import argparse
from datetime import datetime
import codecs
from database.database import Database
from scripts.feed_scraper import EGPFeedScraper

class UTFStreamHandler(logging.StreamHandler):
    def emit(self, record):
        try:
            msg = self.format(record)
            stream = self.stream
            # If stream is stdout/stderr on Windows, encode for cp1252
            if stream in (sys.stdout, sys.stderr):
                try:
                    stream.write(msg + self.terminator)
                except UnicodeEncodeError:
                    # Fall back to replacing unmappable characters
                    stream.write(msg.encode(stream.encoding, errors='replace').decode(stream.encoding) + self.terminator)
            else:
                stream.write(msg + self.terminator)
            self.flush()
        except Exception:
            self.handleError(record)

def setup_logging():
    """Configure logging with UTF-8 support"""
    log_dir = Path("data/logs")
    log_dir.mkdir(parents=True, exist_ok=True)
    
    # Use custom stream handler for console output
    console_handler = UTFStreamHandler(sys.stdout)
    console_handler.setFormatter(logging.Formatter('%(asctime)s - %(levelname)s - %(message)s'))
    
    # Use UTF-8 file handler for log file
    log_file = log_dir / 'egp_scraper.log'
    file_handler = logging.FileHandler(log_file, 'a', encoding='utf-8')
    file_handler.setFormatter(logging.Formatter('%(asctime)s - %(levelname)s - %(message)s'))
    
    # Configure root logger
    root_logger = logging.getLogger()
    root_logger.setLevel(logging.INFO)
    root_logger.addHandler(console_handler)
    root_logger.addHandler(file_handler)

def setup_parser() -> argparse.ArgumentParser:
    """Set up command line argument parser"""
    parser = argparse.ArgumentParser(description='EGP Procurement Data Pipeline')
    subparsers = parser.add_subparsers(dest='command', help='Available commands')
    
    # readfeed command
    read_parser = subparsers.add_parser('readfeed', help='Read EGP RSS feed')
    read_parser.add_argument('dept_id', nargs='?', help='4-digit department code (e.g., 0307)')
    read_parser.add_argument('--dept-sub-id', help='10-digit sub-department code')
    read_parser.add_argument('--method-id', help='2-digit procurement method code (e.g., 16 for e-bidding)')
    read_parser.add_argument('--announce-type', help='2-character announcement type (e.g., P0 for procurement plan)')
    read_parser.add_argument('--date', help='Announcement date in YYYYMMDD format')
    read_parser.add_argument('--count', action='store_true', help='Include count of announcements per day')
    
    # find command
    find_parser = subparsers.add_parser('find', help='Find recent announcements')
    find_parser.add_argument('dept_id', nargs='?', help='4-digit department code (e.g., 0307)')
    find_parser.add_argument('limit', type=int, nargs='?', default=10, help='Number of announcements to show')
    
    # debug command
    debug_parser = subparsers.add_parser('debug', help='Show database contents')
    
    return parser

def process_readfeed(args):
    """Process the readfeed command"""
    try:
        with Database() as db:
            scraper = EGPFeedScraper(db)
            
            # Build parameters dict from args
            params = {
                'dept_id': args.dept_id,
                'dept_sub_id': args.dept_sub_id,
                'method_id': args.method_id,
                'announce_type': args.announce_type,
                'announce_date': args.date,
                'count_by_day': args.count
            }
            
            # Remove None values
            params = {k: v for k, v in params.items() if v is not None}
            
            # Log the parameters being used
            if params:
                logging.info("Fetching feed with parameters:")
                for key, value in params.items():
                    logging.info(f"  {key}: {value}")
            else:
                logging.info("Fetching feed without parameters")
            
            new_entries = scraper.process_feed(**params)
            
            logging.info(f"Feed processing completed. New entries: {new_entries}")
            
    except Exception as e:
        logging.error(f"Error in process_readfeed: {e}")
        raise

def process_find(args):
    """Process the find command"""
    try:
        with Database() as db:
            announcements = db.get_recent_announcements(args.dept_id, args.limit)
            
            if not announcements:
                print("\nNo announcements found in database.")
                return
                
            print(f"\nFound {len(announcements)} recent announcements:")
            print("=" * 100)
            
            for i, ann in enumerate(announcements, 1):
                # Format the announcement for display
                title = ann['title'].strip()
                published = ann['published_date'].strip() if ann['published_date'] else 'N/A'
                created = datetime.fromisoformat(ann['created_at'].replace('Z', '+00:00')).strftime('%Y-%m-%d %H:%M:%S')
                
                print(f"\n{i}. Title: {title}")
                print(f"   Published Date: {published}")
                print(f"   Created in DB: {created}")
                print(f"   Link: {ann['link']}")
                print("-" * 100)
                
    except Exception as e:
        logging.error(f"Error in process_find: {e}")
        raise

def process_debug(args):
    """Debug command to inspect database contents"""
    try:
        with Database() as db:
            db.cursor.execute("SELECT title, description, link FROM announcements")
            results = db.cursor.fetchall()
            
            print(f"\nFound {len(results)} total announcements in database:")
            print("=" * 100)
            
            for i, (title, description, link) in enumerate(results, 1):
                print(f"\n{i}. Title: {title[:150]}...")
                print(f"   Description: {description[:150] if description else 'None'}...")
                print(f"   Link: {link[:150]}...")
                print("-" * 100)
    except Exception as e:
        logging.error(f"Error in process_debug: {e}")
        raise

def main():
    """Main execution function"""
    setup_logging()
    parser = setup_parser()
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        return
    
    logging.info(f"Starting EGP data pipeline - Command: {args.command}")
    
    if args.command == 'readfeed':
        process_readfeed(args)
    elif args.command == 'find':
        process_find(args)
    elif args.command == 'debug':
        process_debug(args)
    else:
        parser.print_help()

if __name__ == "__main__":
    # Ensure UTF-8 is used for stdout/stderr if possible
    if sys.stdout.encoding != 'utf-8':
        if hasattr(sys.stdout, 'reconfigure'):
            sys.stdout.reconfigure(encoding='utf-8')
        if hasattr(sys.stderr, 'reconfigure'):
            sys.stderr.reconfigure(encoding='utf-8')
    main()