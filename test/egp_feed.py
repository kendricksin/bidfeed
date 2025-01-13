import requests
import xml.etree.ElementTree as ET
import csv
from datetime import datetime
import os
import time
import codecs

def fetch_egp_feed(dept_id=None, dept_sub_id=None, announce_type=None):
    """
    Fetch the e-GP RSS feed with optional parameters
    """
    base_url = "http://process3.gprocurement.go.th/EPROCRssFeedWeb/egpannouncerss.xml"
    params = {}
    
    if dept_id:
        params['deptId'] = dept_id
    if dept_sub_id:
        params['deptsubId'] = dept_sub_id
    if announce_type:
        params['anounceType'] = announce_type
        
    headers = {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
        'Accept': 'application/xml',
        'Accept-Language': 'en-US,en;q=0.9,th;q=0.8',
    }
    
    try:
        response = requests.get(
            base_url, 
            params=params, 
            headers=headers,
            timeout=30
        )
        response.encoding = 'cp874'  # Set encoding to Windows-874 (cp874)
        return response.text
    except requests.exceptions.RequestException as e:
        print(f"Error fetching feed: {e}")
        return None

def parse_and_save_feed(content, output_dir="egp_output"):
    """
    Parse the feed and save to CSV and text files
    """
    if not content:
        return
    
    # Create output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)
    
    # Generate timestamp for filenames
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    
    try:
        # Remove any BOM or problematic characters at the start
        content = content.strip()
        if content.startswith('<?xml'):
            content = '<?xml version="1.0" encoding="utf-8"?>' + content[content.find('>')+1:]
        
        root = ET.fromstring(content)
        announcements = []
        
        # Parse each item in the feed
        for item in root.findall('.//item'):
            announcement = {
                'title': item.find('title').text if item.find('title') is not None else '',
                'link': item.find('link').text if item.find('link') is not None else '',
                'description': item.find('description').text if item.find('description') is not None else '',
                'pubDate': item.find('pubDate').text if item.find('pubDate') is not None else ''
            }
            announcements.append(announcement)
        
        # Save as CSV
        csv_filename = os.path.join(output_dir, f'egp_feed_{timestamp}.csv')
        with open(csv_filename, 'w', newline='', encoding='utf-8') as csvfile:
            fieldnames = ['title', 'link', 'description', 'pubDate']
            writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
            writer.writeheader()
            writer.writerows(announcements)
            
        # Save as readable text
        txt_filename = os.path.join(output_dir, f'egp_feed_{timestamp}.txt')
        with open(txt_filename, 'w', encoding='utf-8') as txtfile:
            txtfile.write("e-GP Procurement Announcements\n")
            txtfile.write("=" * 50 + "\n\n")
            
            for ann in announcements:
                txtfile.write("Title: " + ann['title'] + "\n")
                txtfile.write("Link: " + ann['link'] + "\n")
                txtfile.write("Published: " + ann['pubDate'] + "\n")
                txtfile.write("Description: " + ann['description'] + "\n")
                txtfile.write("-" * 50 + "\n\n")
                
        print(f"Files saved successfully:")
        print(f"CSV: {csv_filename}")
        print(f"Text: {txt_filename}")
        
    except ET.ParseError as e:
        print(f"Error parsing XML: {e}")
        print("Raw content:")
        print(content[:500])  # Print first 500 characters for debugging
    except Exception as e:
        print(f"Error processing feed: {e}")

def main():
    # Example usage with the Comptroller General's Department (dept_id = 0304)
    dept_id = "03"
    print(f"Fetching e-GP feed for department ID: {dept_id}")
    
    current_hour = datetime.now().hour
    current_minute = datetime.now().minute
    
    # Check if current time is within allowed periods
    is_allowed_time = (
        (12 <= current_hour < 13) or
        (17 <= current_hour <= 23) or
        (0 <= current_hour <= 8)
    )
    
    if not is_allowed_time:
        print("\nWarning: Current time is outside the allowed access periods:")
        print("- 12:01 - 12:59")
        print("- 17:01 - 08:59")
        print("The request might fail.")
    
    content = fetch_egp_feed(dept_id=dept_id)
    if content:
        parse_and_save_feed(content)

if __name__ == "__main__":
    main()