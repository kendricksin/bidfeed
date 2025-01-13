import PyPDF2
import re
from pathlib import Path

class PDFExtractor:
    def __init__(self):
        self.thai_to_arabic = str.maketrans('๐๑๒๓๔๕๖๗๘๙', '0123456789')

    def convert_thai_number(self, thai_number):
        """Convert Thai numerals to Arabic numerals"""
        return thai_number.translate(self.thai_to_arabic)

    def extract_budget(self, text):
        """Extract budget amount from text"""
        # Look for numbers followed by บาท
        pattern = r'([\d,]+\.?\d*)\s*บาท'
        match = re.search(pattern, text)
        if match:
            amount = match.group(1)
            return {
                'amount': amount,
                'amount_clean': amount.replace(',', '')
            }
        return None

    def extract_quantity_specs(self, text):
        """Extract quantity specifications"""
        pattern = r'จำนวน\s*(\d+)'
        matches = re.findall(pattern, text)
        if matches:
            return matches[0]  # Return first match
        return None

    def extract_duration(self, text):
        """Extract contract duration"""
        year_pattern = r'ระยะเวลา\s*(\d+)\s*ปี'
        month_pattern = r'\((\d+)\s*เดือน\)'
        
        duration = {}
        year_match = re.search(year_pattern, text)
        month_match = re.search(month_pattern, text)
        
        if year_match:
            duration['years'] = year_match.group(1)
        if month_match:
            duration['months'] = month_match.group(1)
        return duration if duration else None

    def extract_submission_info(self, text):
        """Extract submission date and time"""
        # Looking for dates with Thai month names
        date_pattern = r'วันที่\s*(\d+.*\d{4})'
        time_pattern = r'(\d{2}[:\.]\d{2})\s*น'
        
        submission_info = {}
        date_match = re.search(date_pattern, text)
        time_match = re.search(time_pattern, text)
        
        if date_match:
            submission_info['date'] = date_match.group(1).strip()
        if time_match:
            submission_info['time'] = time_match.group(1)
        return submission_info if submission_info else None

    def extract_contact_info(self, text):
        """Extract contact information"""
        phone_pattern = r'โทรศัพท์.*?(\d[\d\-]+)'
        email_pattern = r'([a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)'
        
        contact_info = {}
        phone_match = re.search(phone_pattern, text)
        email_match = re.search(email_pattern, text)
        
        if phone_match:
            contact_info['phone'] = phone_match.group(1)
        if email_match:
            contact_info['email'] = email_match.group(1)
        return contact_info if contact_info else None

    def parse_pdf(self, pdf_path):
        """Parse PDF and extract key information"""
        try:
            with open(pdf_path, 'rb') as file:
                reader = PyPDF2.PdfReader(file)
                full_text = ''
                
                # Print each page text for debugging
                print("\nExtracting text from PDF pages:")
                for i, page in enumerate(reader.pages):
                    page_text = page.extract_text()
                    print(f"\nPage {i+1}:")
                    print("-" * 30)
                    print(page_text[:200] + "...")  # Print first 200 chars of each page
                    full_text += page_text + '\n'

                # Extract all information
                info = {
                    'budget': self.extract_budget(full_text),
                    'specifications': self.extract_quantity_specs(full_text),
                    'duration': self.extract_duration(full_text),
                    'submission_info': self.extract_submission_info(full_text),
                    'contact_info': self.extract_contact_info(full_text),
                }
                
                return info
        except Exception as e:
            print(f"Error parsing PDF: {e}")
            return None

def main():
    extractor = PDFExtractor()
    
    pdf_path = "sample.pdf"
    results = extractor.parse_pdf(pdf_path)
    
    if results:
        print("\nExtracted Information:")
        print("=" * 50)
        
        if results['budget']:
            # Convert Thai numerals to Arabic
            thai_amount = results['budget']['amount']
            arabic_amount = thai_amount.translate(extractor.thai_to_arabic)
            clean_amount = arabic_amount.replace(',', '')
            print(f"\nBudget:")
            print(f"Amount: {arabic_amount} บาท")
            print(f"Clean amount: {float(clean_amount):,.2f} บาท")
        
        if results['specifications']:
            quantity = results['specifications'].translate(extractor.thai_to_arabic)
            print(f"\nQuantity: {quantity}")
        
        if results['duration']:
            print(f"\nDuration:")
            if 'years' in results['duration']:
                years = results['duration']['years'].translate(extractor.thai_to_arabic)
                print(f"- Years: {years}")
            if 'months' in results['duration']:
                months = results['duration']['months'].translate(extractor.thai_to_arabic)
                print(f"- Months: {months}")
        
        if results['submission_info']:
            print(f"\nSubmission Information:")
            if 'date' in results['submission_info']:
                date = results['submission_info']['date']
                # Convert only the numbers in the date, keeping Thai month name
                date_parts = []
                for part in date.split():
                    if any(c in '๐๑๒๓๔๕๖๗๘๙' for c in part):
                        date_parts.append(part.translate(extractor.thai_to_arabic))
                    else:
                        date_parts.append(part)
                converted_date = ' '.join(date_parts)
                print(f"- Date: {converted_date}")
            if 'time' in results['submission_info']:
                time = results['submission_info']['time'].translate(extractor.thai_to_arabic)
                print(f"- Time: {time}")
        
        if results['contact_info']:
            print(f"\nContact Information:")
            if 'phone' in results['contact_info']:
                phone = results['contact_info']['phone'].translate(extractor.thai_to_arabic)
                print(f"- Phone: {phone}")
            if 'email' in results['contact_info']:
                email = results['contact_info']['email']
                print(f"- Email: {email}")

if __name__ == "__main__":
    main()