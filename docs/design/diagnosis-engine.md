# Diagnosis Engine Design

## Overview
The diagnosis engine analyzes middleware state using rules and AI.

## Architecture
1. Plugin collection
2. Rule evaluation (Rule Engine)
3. Advanced analysis (Analyzers)
4. Report generation

## Rule DSL
Uses `expr` language for conditions.
Examples: `metrics.cpu > 80`, `len(slowlogs) > 10`.
