import logging
import sys
from pathlib import Path
from typing import Optional, Dict, List
import requests
import xml.etree.ElementTree as ET
from datetime import datetime
import time

# Add parent directory to Python path
sys.path.append(str(Path(__file__).parent.parent))

from database.database import Database

class EGPFeedScraper:
    def __init__(self, db: Database):
        self.db = db
        self.base_url = "http://process3.gprocurement.go.th/EPROCRssFeedWeb/egpannouncerss.xml"
        
    def fetch_feed(self, 
                  dept_id: Optional[str] = None,
                  dept_sub_id: Optional[str] = None,
                  method_id: Optional[str] = None,
                  announce_type: Optional[str] = None,
                  announce_date: Optional[str] = None,
                  count_by_day: bool = False) -> Optional[str]:
        """
        Fetch the e-GP RSS feed with optional parameters
        
        Args:
            dept_id: 4-digit department code (e.g., "0307" for Revenue Department)
            dept_sub_id: 10-digit sub-department code
            method_id: 2-digit procurement method code (e.g., "16" for e-bidding)
            announce_type: 2-character announcement type (e.g., "P0" for procurement plan)
            announce_date: Date in YYYYMMDD format
            count_by_day: Whether to include count of announcements per day
        """
        params = {}
        if dept_id:
            params['deptId'] = dept_id
        if dept_sub_id:
            params['deptsubId'] = dept_sub_id
        if method_id:
            params['methodId'] = method_id
        if announce_type:
            params['anounceType'] = announce_type
        if announce_date:
            params['announceDate'] = announce_date
        if count_by_day:
            params['countbyday'] = ""
            
        headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
            'Accept': 'application/xml',
            'Accept-Language': 'en-US,en;q=0.9,th;q=0.8',
        }

        # Check if current time is within allowed periods
        current_hour = datetime.now().hour
        current_minute = datetime.now().minute
        
        is_allowed_time = (
            (12 <= current_hour < 13) or  # 12:01 - 12:59
            (17 <= current_hour <= 23) or  # 17:01 - 23:59
            (0 <= current_hour <= 8)       # 00:00 - 08:59
        )
        
        if not is_allowed_time:
            logging.warning("Current time is outside the allowed access periods:")
            logging.warning("- 12:01 - 12:59")
            logging.warning("- 17:01 - 08:59")
            logging.warning("The request might fail.")
        
        try:
            response = requests.get(
                self.base_url,
                params=params,
                headers=headers,
                timeout=30
            )
            response.encoding = 'cp874'  # Set encoding to Windows-874
            
            if response.status_code != 200:
                logging.error(f"Failed to fetch feed. Status code: {response.status_code}")
                return None
                
            return response.text
        except requests.exceptions.RequestException as e:
            logging.error(f"Error fetching feed: {e}")
            return None
            
    def parse_feed(self, content: str) -> List[Dict]:
        """Parse the XML feed content and return a list of announcements"""
        if not content:
            return []
            
        try:
            # Remove any BOM or problematic characters
            content = content.strip()
            if content.startswith('<?xml'):
                content = '<?xml version="1.0" encoding="utf-8"?>' + content[content.find('>')+1:]
            
            root = ET.fromstring(content)
            announcements = []
            
            # Get countbyday if present
            countbyday = root.find('.//countbyday')
            if countbyday is not None:
                logging.info(f"Total announcements for today: {countbyday.text}")
            
            for item in root.findall('.//item'):
                announcement = {
                    'title': item.find('title').text if item.find('title') is not None else '',
                    'link': item.find('link').text if item.find('link') is not None else '',
                    'description': item.find('description').text if item.find('description') is not None else '',
                    'published_date': item.find('pubDate').text if item.find('pubDate') is not None else ''
                }
                announcements.append(announcement)
                
            return announcements
        except ET.ParseError as e:
            logging.error(f"Error parsing XML: {e}")
            logging.debug(f"Problematic content: {content[:500]}")
            return []
            
    def process_feed(self, **kwargs) -> int:
        """
        Process the feed and store in database
        Returns the number of new announcements processed
        """
        content = self.fetch_feed(**kwargs)
        if not content:
            return 0
            
        announcements = self.parse_feed(content)
        
        if announcements:
            # Log the first announcement for verification
            first_announcement = announcements[0]
            logging.info("First announcement details:")
            logging.info(f"Title: {first_announcement['title']}")
            logging.info(f"Link: {first_announcement['link']}")
            logging.info(f"Published: {first_announcement['published_date']}")
        
        # Store announcements in database
        new_entries = 0
        dept_id = kwargs.get('dept_id')  # Get department ID from request parameters
        for announcement in announcements:
            try:
                announcement_id = self.db.insert_announcement(announcement, dept_id)
                if announcement_id:
                    new_entries += 1
            except Exception as e:
                logging.error(f"Error storing announcement: {e}")
                continue
                
        logging.info(f"Total announcements found: {len(announcements)}")
        logging.info(f"New announcements stored: {new_entries}")
        
        return new_entries