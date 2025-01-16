import logging
import asyncio
import aiohttp
import os
import ssl
from pathlib import Path
from typing import List, Dict, Optional
import re
from urllib.parse import unquote

class PDFDownloader:
    def __init__(self, output_dir: str = "data/project_docs"):
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)
        
    async def download_pdf(self, url: str, project_id: str) -> Optional[str]:
        """Download a single PDF file"""
        try:
            # Create project directory
            project_dir = self.output_dir / project_id
            project_dir.mkdir(exist_ok=True)
            
            # Extract filename from URL or use project_id if not available
            filename = unquote(url.split('/')[-1])
            if not filename.endswith('.pdf'):
                filename = f"{project_id}.pdf"
            
            # Clean filename of invalid characters
            filename = re.sub(r'[<>:"/\\|?*]', '_', filename)
            filepath = project_dir / filename
            
            # Skip if file already exists
            if filepath.exists():
                logging.info(f"File already exists: {filepath}")
                return str(filepath)

            # Set up browser-like headers
            headers = {
                'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
                'Accept': 'text/html,application/xhtml+xml,application/xml,application/pdf',
                'Accept-Language': 'en-US,en;q=0.5,th;q=0.3',
                'Connection': 'keep-alive',
            }

            # SSL context that skips verification
            ssl_context = ssl.create_default_context()
            ssl_context.check_hostname = False
            ssl_context.verify_mode = ssl.CERT_NONE

            connector = aiohttp.TCPConnector(ssl=ssl_context)

            async with aiohttp.ClientSession(connector=connector) as session:
                try:
                    logging.info(f"Attempting to download from: {url}")
                    async with session.get(url, headers=headers, allow_redirects=True) as response:
                        if response.status != 200:
                            logging.error(f"Failed download: HTTP {response.status}")
                            return None

                        # Log response details for debugging
                        logging.info(f"Response headers: {dict(response.headers)}")
                        
                        # Download the file
                        with open(filepath, 'wb') as f:
                            async for chunk in response.content.iter_chunked(8192):
                                f.write(chunk)
                        
                        # Verify the file is a PDF
                        if os.path.getsize(filepath) > 0:
                            with open(filepath, 'rb') as f:
                                if f.read(4).startswith(b'%PDF'):
                                    logging.info(f"Successfully downloaded: {filepath}")
                                    return str(filepath)
                                else:
                                    os.remove(filepath)
                                    logging.error("Downloaded file is not a valid PDF")
                                    return None
                        else:
                            os.remove(filepath)
                            logging.error("Downloaded file is empty")
                            return None
                            
                except Exception as e:
                    logging.error(f"Error during download attempt: {str(e)}")
                    return None

        except Exception as e:
            logging.error(f"Error in download process: {str(e)}")
            return None
            
    async def download_batch(self, announcements: List[Dict]) -> List[Dict]:
        """Download PDFs for multiple announcements"""
        results = []
        
        for announcement in announcements:
            project_id = announcement.get('project_id', 'unknown')
            url = announcement.get('link')
            
            if not url:
                logging.warning(f"No URL found for project {project_id}")
                continue
                
            filepath = await self.download_pdf(url, project_id)
            
            results.append({
                'project_id': project_id,
                'url': url,
                'filepath': filepath,
                'success': filepath is not None
            })
            
        return results

def download_pdfs(announcements: List[Dict]) -> List[Dict]:
    """Synchronous wrapper for PDF downloads"""
    downloader = PDFDownloader()
    return asyncio.run(downloader.download_batch(announcements))