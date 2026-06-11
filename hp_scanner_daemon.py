#!/usr/bin/env python3
"""
HP Smart Tank 750 ADF 自动扫描守护程序
当 ADF（自动输稿器）有纸时自动触发扫描
扫描文件保存到 ~/测试图片/<开始时间>/
"""

import os
import sys
import time
import signal
import requests
import xml.etree.ElementTree as ET
from datetime import datetime
from pathlib import Path

# 配置
SCANNER_IP = "192.168.3.11"
BASE_URL = f"http://{SCANNER_IP}"
OUTPUT_BASE = Path("/Users/wangyou/测试图片")
POLL_INTERVAL = 2  # 检查 ADF 状态的间隔（秒）
ESCL_TIMEOUT = 10  # eSCL 请求超时

# 命名空间
NS = {
    'scan': 'http://schemas.hp.com/imaging/escl/2011/05/03',
    'pwg': 'http://www.pwg.org/schemas/2010/12/sm'
}

class ScannerDaemon:
    def __init__(self):
        self.running = False
        self.scanning = False  # 防止重复扫描的锁
        self.current_job = None
        self.output_dir = None
        self.page_count = 0
        self.last_adf_empty = True  # 记录上次 ADF 是否为空，用于检测纸张放入

    def log(self, msg):
        """打印带时间戳的日志"""
        print(f"[{datetime.now().strftime('%H:%M:%S')}] {msg}", flush=True)

    def check_adf_status(self):
        """检查 ADF 是否有纸"""
        try:
            resp = requests.get(
                f"{BASE_URL}/eSCL/ScannerStatus",
                timeout=ESCL_TIMEOUT
            )
            resp.raise_for_status()

            root = ET.fromstring(resp.content)
            adf_state = root.find('.//scan:AdfState', NS)
            scanner_state = root.find('.//pwg:State', NS)

            adf_text = adf_state.text if adf_state is not None else "Unknown"
            scanner_text = scanner_state.text if scanner_state is not None else "Unknown"

            return {
                'has_paper': adf_text not in ['ScannerAdfEmpty', 'Empty', 'AdfEmpty'],
                'adf_state': adf_text,
                'scanner_state': scanner_text
            }

        except Exception as e:
            self.log(f"检查 ADF 状态失败: {e}")
            return {'has_paper': False, 'adf_state': 'Error', 'scanner_state': 'Error'}

    def create_scan_job(self, resolution=300, color_mode="RGB24", format="image/jpeg"):
        """创建 eSCL 扫描任务 - 强制使用JPEG格式输出单张图片"""
        settings_xml = f"""<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
    <pwg:Version>2.63</pwg:Version>
    <scan:Intent>Document</scan:Intent>
    <scan:InputSource>Feeder</scan:InputSource>
    <scan:ColorMode>{color_mode}</scan:ColorMode>
    <scan:XResolution>{resolution}</scan:XResolution>
    <scan:YResolution>{resolution}</scan:YResolution>
    <pwg:DocumentFormat>image/jpeg</pwg:DocumentFormat>
    <scan:DocumentFormatExt>image/jpeg</scan:DocumentFormatExt>
    <scan:ContentType>Text</scan:ContentType>
</scan:ScanSettings>"""

        try:
            resp = requests.post(
                f"{BASE_URL}/eSCL/ScanJobs",
                data=settings_xml,
                headers={'Content-Type': 'application/xml'},
                timeout=ESCL_TIMEOUT
            )

            if resp.status_code == 201:
                location = resp.headers.get('Location', '')
                job_id = location.split('/')[-1] if location else None
                return job_id
            else:
                self.log(f"创建扫描任务失败: HTTP {resp.status_code}")
                return None

        except Exception as e:
            self.log(f"创建扫描任务异常: {e}")
            return None

    def wait_for_pages(self, job_id, timeout=300):
        """等待扫描完成并下载所有页面"""
        start_time = time.time()
        downloaded = 0
        last_status_code = None
        status_error_count = 0

        self.log(f"开始监控任务 {job_id[:8]}...")

        while time.time() - start_time < timeout:
            try:
                # 检查任务状态
                status_resp = requests.get(
                    f"{BASE_URL}/eSCL/ScanJobs/{job_id}",
                    timeout=ESCL_TIMEOUT
                )

                if status_resp.status_code != 200:
                    status_error_count += 1
                    if status_resp.status_code != last_status_code:
                        self.log(f"任务状态查询失败: HTTP {status_resp.status_code}")
                        last_status_code = status_resp.status_code

                    # 如果连续多次失败，任务可能已消失
                    if status_error_count > 5:
                        self.log(f"任务查询连续失败 {status_error_count} 次，可能已结束")
                        # 尝试直接下载看是否还有页面
                        for page_num in range(downloaded + 1, downloaded + 10):
                            if self.download_page(job_id, page_num):
                                downloaded += 1
                                self.log(f"已下载第 {page_num} 页")
                            else:
                                break
                        return downloaded

                    time.sleep(1)
                    continue

                # 重置错误计数
                status_error_count = 0
                last_status_code = 200

                try:
                    root = ET.fromstring(status_resp.content)
                except ET.ParseError as e:
                    self.log(f"XML解析错误: {e}")
                    time.sleep(1)
                    continue

                job_state = root.find('.//pwg:JobState', NS)
                images_completed = root.find('.//pwg:ImagesCompleted', NS)
                images_to_transfer = root.find('.//pwg:ImagesToTransfer', NS)

                completed = int(images_completed.text) if images_completed is not None else 0
                to_transfer = int(images_to_transfer.text) if images_to_transfer is not None else 0
                state = job_state.text if job_state is not None else "Unknown"

                # 每10秒或状态变化时记录一次
                elapsed = int(time.time() - start_time)
                if elapsed % 10 == 0 or state not in ['Processing', 'Pending']:
                    self.log(f"任务状态: {state}, 已完成: {completed}, 待传输: {to_transfer}")

                # 下载新完成的页面
                while downloaded < completed:
                    page_num = downloaded + 1
                    if self.download_page(job_id, page_num):
                        downloaded += 1
                        self.log(f"✅ 已下载第 {page_num} 页")
                    else:
                        self.log(f"❌ 下载第 {page_num} 页失败")
                        break

                # 检查是否完成
                if state in ['Completed', 'Canceled', 'Aborted']:
                    self.log(f"任务结束，状态: {state}")
                    # 下载剩余页面
                    while downloaded < completed:
                        page_num = downloaded + 1
                        if self.download_page(job_id, page_num):
                            downloaded += 1
                        else:
                            break
                    return downloaded

                time.sleep(0.5)

            except requests.exceptions.RequestException as e:
                self.log(f"网络错误: {e}")
                time.sleep(2)
            except Exception as e:
                self.log(f"等待页面时出错: {e}")
                import traceback
                self.log(traceback.format_exc())
                time.sleep(1)

        self.log(f"等待超时 ({timeout}秒)")
        return downloaded

    def download_page(self, job_id, page_num):
        """下载指定页面 - 支持PDF和多页拆分"""
        try:
            resp = requests.get(
                f"{BASE_URL}/eSCL/ScanJobs/{job_id}/NextDocument",
                timeout=ESCL_TIMEOUT,
                stream=True
            )

            if resp.status_code == 200:
                content_type = resp.headers.get('Content-Type', '')

                # 根据内容类型处理
                if 'pdf' in content_type:
                    # 保存临时PDF文件
                    temp_pdf = self.output_dir / f"temp_{page_num:03d}.pdf"
                    with open(temp_pdf, 'wb') as f:
                        for chunk in resp.iter_content(chunk_size=8192):
                            f.write(chunk)

                    # 使用pdf2image转换为多页JPG
                    try:
                        from pdf2image import convert_from_path
                        images = convert_from_path(temp_pdf, dpi=300)

                        # 为每一页生成单独的JPG文件
                        for idx, image in enumerate(images, start=1):
                            # 如果是多页PDF，使用 page_XXX_YY.jpg 格式
                            if len(images) > 1:
                                jpg_filename = self.output_dir / f"page_{page_num:03d}_{idx:02d}.jpg"
                            else:
                                jpg_filename = self.output_dir / f"page_{page_num:03d}.jpg"

                            image.save(jpg_filename, 'JPEG', quality=95)
                            self.log(f"✅ 已保存: {jpg_filename.name} ({image.width}x{image.height})")

                        # 删除临时PDF
                        temp_pdf.unlink()

                    except Exception as e:
                        self.log(f"PDF转换失败: {e}")
                        return False
                else:
                    # 直接保存为JPG（图片格式）
                    jpg_filename = self.output_dir / f"page_{page_num:03d}.jpg"
                    with open(jpg_filename, 'wb') as f:
                        for chunk in resp.iter_content(chunk_size=8192):
                            f.write(chunk)

                    # 验证并转换格式
                    try:
                        from PIL import Image
                        img = Image.open(jpg_filename)
                        if img.format != 'JPEG':
                            img.convert('RGB').save(jpg_filename, 'JPEG', quality=95)
                            self.log(f"已转换为JPEG: {jpg_filename.name}")
                    except Exception as e:
                        self.log(f"图片处理警告: {e}")

                return True
            else:
                return False

        except Exception as e:
            self.log(f"下载第 {page_num} 页失败: {e}")
            return False

    def run_scan_session(self):
        """运行一次完整的扫描会话"""
        self.scanning = True
        job_id = None

        try:
            # 创建以当前时间命名的输出目录
            timestamp = datetime.now().strftime("%Y-%m-%d_%H.%M.%S")
            self.output_dir = OUTPUT_BASE / timestamp
            self.output_dir.mkdir(parents=True, exist_ok=True)

            self.log(f"=" * 50)
            self.log(f"开始扫描会话，输出目录: {self.output_dir.name}")
            self.log(f"格式: JPEG (支持连续扫描，每页单独文件)")
            self.log(f"=" * 50)

            # 创建扫描任务（带重试）
            job_id = None
            for attempt in range(5):  # 最多重试5次
                job_id = self.create_scan_job(resolution=300, color_mode="RGB24")
                if job_id:
                    break
                self.log(f"第 {attempt+1}/5 次创建任务失败，2秒后重试...")
                time.sleep(2)

            if not job_id:
                self.log("❌ 创建扫描任务失败，放弃")
                try:
                    self.output_dir.rmdir()  # 删除空目录
                except:
                    pass
                return False

            self.log(f"✅ 扫描任务已创建: {job_id}")

            # 等待并下载所有页面
            pages = self.wait_for_pages(job_id)
            self.log(f"📄 扫描完成，共 {pages} 页")

            # 重置 ADF 状态检测（扫描完成后 ADF 应该变空）
            self.last_adf_empty = True

            if pages == 0:
                try:
                    self.output_dir.rmdir()  # 删除空目录
                except:
                    pass
                self.log("⚠️ 没有扫描到任何页面，可能 ADF 中没有纸或纸张放置不当")
            else:
                self.log(f"✅ 扫描成功！文件保存在: {self.output_dir}")

            return pages > 0

        except Exception as e:
            self.log(f"❌ 扫描会话异常: {e}")
            import traceback
            self.log(traceback.format_exc())
            return False
        finally:
            self.scanning = False
            # 清理任务
            if job_id:
                try:
                    requests.delete(f"{BASE_URL}/eSCL/ScanJobs/{job_id}", timeout=5)
                    self.log(f"🧹 已清理任务 {job_id[:8]}")
                except:
                    pass

    def run(self):
        """主循环"""
        self.running = True
        self.log("=" * 50)
        self.log("HP Smart Tank 750 ADF 自动扫描守护程序已启动")
        self.log(f"扫描仪: {SCANNER_IP}")
        self.log(f"输出目录: {OUTPUT_BASE}")
        self.log("等待 ADF 有纸...")
        self.log("=" * 50)

        while self.running:
            try:
                status = self.check_adf_status()

                # 边缘检测：从空到非空（纸张放入）
                paper_inserted = status['has_paper'] and self.last_adf_empty

                # 关键：检测到有纸放入且未在扫描中时触发
                # 不严格要求扫描仪状态为 Idle，因为放入纸张后状态可能立即变为 Processing
                if paper_inserted and not self.scanning:
                    self.log(f"🟢 检测到纸张放入 ADF: {status['adf_state']}")

                    # 等待扫描仪完全准备好（避免 503 错误）
                    self.log("等待扫描仪准备就绪...")
                    ready = False
                    for retry in range(10):  # 最多等5秒
                        time.sleep(0.5)
                        check_status = self.check_adf_status()
                        # 扫描仪可能从 Processing 变为 Idle，或者有纸就行
                        if check_status['has_paper']:
                            ready = True
                            break

                    if ready:
                        self.run_scan_session()
                    else:
                        self.log("⚠️ 扫描仪未就绪，跳过本次扫描")

                    self.log("等待 ADF 有纸...")

                # 更新上次状态
                self.last_adf_empty = not status['has_paper']

                time.sleep(POLL_INTERVAL)

            except KeyboardInterrupt:
                self.log("收到中断信号，正在退出...")
                break
            except Exception as e:
                self.log(f"主循环异常: {e}")
                time.sleep(POLL_INTERVAL)

    def stop(self):
        """停止守护程序"""
        self.running = False

def main():
    daemon = ScannerDaemon()

    def signal_handler(signum, frame):
        daemon.stop()

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    # 确保输出目录存在
    OUTPUT_BASE.mkdir(parents=True, exist_ok=True)

    daemon.run()
    print("守护程序已退出")

if __name__ == "__main__":
    main()
