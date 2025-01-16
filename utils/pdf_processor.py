import logging
import asyncio
from datetime import datetime
from pathlib import Path
from typing import List, Dict, Optional
from database.database import Database
from utils.pdf_download import download_pdfs
from utils.pdf_extractor import PDFExtractor

class PDFProcessor:
    def __init__(self, db: Database):
        self.db = db
        self.extractor = PDFExtractor()
        
    def process_pdf_data(self, pdf_path: str, announcement_id: int) -> bool:
        """Process a single PDF and store its data"""
        try:
            # Extract data from PDF
            logging.info(f"Extracting data from {pdf_path}")
            extracted_data = self.extractor.parse_pdf(pdf_path)
            
            if not extracted_data:
                logging.error(f"No data extracted from {pdf_path}")
                return False
            
            # Prepare data for database
            procurement_data = {
                'announcement_id': announcement_id,
                'budget_amount': None,
                'quantity': None,
                'duration_years': None,
                'duration_months': None,
                'submission_date': None,
                'submission_time': None,
                'contact_phone': None,
                'contact_email': None,
                'extracted_at': datetime.now()
            }
            
            # Budget
            if extracted_data.get('budget'):
                try:
                    clean_amount = extracted_data['budget']['amount_clean']
                    procurement_data['budget_amount'] = float(clean_amount)
                except (ValueError, KeyError) as e:
                    logging.warning(f"Could not parse budget amount: {e}")
            
            # Quantity
            if extracted_data.get('specifications'):
                try:
                    procurement_data['quantity'] = int(extracted_data['specifications'])
                except ValueError as e:
                    logging.warning(f"Could not parse quantity: {e}")
            
            # Duration
            if extracted_data.get('duration'):
                duration = extracted_data['duration']
                if 'years' in duration:
                    try:
                        procurement_data['duration_years'] = int(duration['years'])
                    except ValueError:
                        logging.warning("Could not parse duration years")
                if 'months' in duration:
                    try:
                        procurement_data['duration_months'] = int(duration['months'])
                    except ValueError:
                        logging.warning("Could not parse duration months")
            
            # Submission info
            if extracted_data.get('submission_info'):
                submission = extracted_data['submission_info']
                if 'date' in submission:
                    procurement_data['submission_date'] = submission['date']
                if 'time' in submission:
                    procurement_data['submission_time'] = submission['time']
            
            # Contact info
            if extracted_data.get('contact_info'):
                contact = extracted_data['contact_info']
                procurement_data['contact_phone'] = contact.get('phone')
                procurement_data['contact_email'] = contact.get('email')
            
            # Insert into database
            self.insert_procurement_details(procurement_data)
            logging.info(f"Successfully processed and stored data for announcement {announcement_id}")
            return True
            
        except Exception as e:
            logging.error(f"Error processing PDF {pdf_path}: {e}")
            return False
    
    def insert_procurement_details(self, data: Dict) -> Optional[int]:
        """Insert procurement details into database"""
        try:
            placeholders = ', '.join('?' * len(data))
            columns = ', '.join(data.keys())
            values = tuple(data.values())
            
            query = f"""
                INSERT INTO procurement_details 
                ({columns})
                VALUES ({placeholders})
            """
            
            self.db.cursor.execute(query, values)
            self.db.conn.commit()
            return self.db.cursor.lastrowid
            
        except Exception as e:
            logging.error(f"Error inserting procurement details: {e}")
            return None

def process_announcements(db: Database, dept_id: Optional[str] = None, limit: int = 10):
    """Process announcements: download PDFs and extract data"""
    try:
        # Get announcements
        announcements = db.get_recent_announcements(dept_id, limit)
        if not announcements:
            logging.info("No announcements found to process")
            return
        
        # Download PDFs
        logging.info(f"Downloading PDFs for {len(announcements)} announcements...")
        download_results = download_pdfs(announcements)
        
        # Process downloaded PDFs
        processor = PDFProcessor(db)
        success_count = 0
        
        for result in download_results:
            if not result['success']:
                logging.warning(f"Skipping extraction for failed download: {result['project_id']}")
                continue
                
            # Find corresponding announcement
            announcement = next(
                (a for a in announcements if a['project_id'] == result['project_id']), 
                None
            )
            
            if not announcement:
                logging.warning(f"Could not find announcement for project {result['project_id']}")
                continue
            
            # Process the PDF
            success = processor.process_pdf_data(result['filepath'], announcement['id'])
            if success:
                success_count += 1
        
        logging.info(f"Processing completed. Successfully processed {success_count} of {len(download_results)} PDFs")
        
    except Exception as e:
        logging.error(f"Error in process_announcements: {e}")
        raise

if __name__ == "__main__":
    # Setup logging
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(levelname)s - %(message)s'
    )
    
    # Example usage
    with Database() as db:
        process_announcements(db, dept_id="0307", limit=5)