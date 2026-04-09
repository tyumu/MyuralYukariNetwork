.PHONY: help install dev dev-watch test clean

help:
	@echo "MemU Sidecar API - Makefile commands"
	@echo "  make install      - Install dependencies"
	@echo "  make dev          - Run dev server"
	@echo "  make dev-watch    - Run with auto-reload"
	@echo "  make test         - Run sidecar checks"
	@echo "  make clean        - Clean cache/build files"

install:
	pip install -r requirements.txt

dev:
	python main.py

dev-watch:
	python main.py

test:
	python -m py_compile main.py src/config/__init__.py src/contracts.py src/logger/__init__.py
	python -m unittest discover -s tests -p "test_*.py"

clean:
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete
	rm -rf .pytest_cache .ruff_cache
