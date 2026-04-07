"""Simple logging setup for the sidecar API."""

import logging


class DevLogger:
    """Simple logger with optional dev log buffering."""
    
    def __init__(self, name: str, level: str = "info", dev_mode: bool = False):
        self.logger = logging.getLogger(name)
        
        # Set level
        level_map = {
            "debug": logging.DEBUG,
            "info": logging.INFO,
            "warn": logging.WARNING,
            "error": logging.ERROR,
        }
        self.logger.setLevel(level_map.get(level.lower(), logging.INFO))
        
        # Console handler
        handler = logging.StreamHandler()
        formatter = logging.Formatter(
            "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
        )
        handler.setFormatter(formatter)
        if not self.logger.handlers:
            self.logger.addHandler(handler)
        
        self.dev_mode = dev_mode
        self.log_buffer: list[str] = []
        self.max_buffer = 100
    
    def debug(self, msg: str, **kwargs):
        self.logger.debug(f"{msg} {kwargs}")
        if self.dev_mode:
            self._buffer(f"DEBUG: {msg}")
    
    def info(self, msg: str, **kwargs):
        self.logger.info(f"{msg} {kwargs}")
        if self.dev_mode:
            self._buffer(f"INFO: {msg}")
    
    def warn(self, msg: str, **kwargs):
        self.logger.warning(f"{msg} {kwargs}")
        if self.dev_mode:
            self._buffer(f"WARN: {msg}")
    
    def error(self, msg: str, **kwargs):
        self.logger.error(f"{msg} {kwargs}")
        if self.dev_mode:
            self._buffer(f"ERROR: {msg}")
    
    def _buffer(self, msg: str):
        self.log_buffer.append(msg)
        if len(self.log_buffer) > self.max_buffer:
            self.log_buffer.pop(0)
    
    def get_recent_logs(self, count: int) -> list[str]:
        """Get the most recent log entries."""
        return self.log_buffer[-count:] if self.log_buffer else []


def setup_logger(level: str = "info", dev_mode: bool = False) -> DevLogger:
    """Setup and return a configured logger."""
    return DevLogger("memu-sidecar", level=level, dev_mode=dev_mode)
