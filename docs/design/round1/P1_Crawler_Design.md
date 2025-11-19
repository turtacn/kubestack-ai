# P1: Crawler Design Document

## 1. Overview

This document outlines the design of the web crawler for the KubeStack-AI knowledge base. The crawler is responsible for fetching, cleaning, and structuring content from various web sources, starting with the official documentation for Redis, Kafka, and MySQL.

## 2. Architecture

The crawler is composed of the following components:

- **`WebCrawler` Interface:** Defines the contract for crawling operations.
- **`advancedCrawler`:** The primary implementation of the `WebCrawler` interface, built on the Colly library.
- **`ContentCleaner`:** Responsible for cleaning raw HTML and converting it to Markdown.
- **`QualityScorer`:** Scores the quality of the crawled content based on a set of heuristics.
- **`DocClassifier`:** Classifies the type of document (e.g., tutorial, reference).
- **Configuration:** A YAML file (`configs/knowledge/crawler.yaml`) defines the crawler's behavior, including target sites, concurrency, and filtering rules.

## 3. Detailed Design

### 3.1. `WebCrawler` Interface

```go
type WebCrawler interface {
    Crawl(startURL string) (*store.RawDocument, error)
    CrawlWithFilter(ctx context.Context, startURL string) (<-chan *CrawledDocument, error)
    GetMetadata(doc *CrawledDocument) map[string]string
}
```

### 3.2. `advancedCrawler`

The `advancedCrawler` uses the Colly library for robust and efficient crawling. It features:

- **Concurrency:** The `MaxConcurrency` setting in `crawler.yaml` controls the number of concurrent requests.
- **Filtering:** URL blacklists and whitelists are supported to control the crawl scope.
- **Deduplication:** A `sync.Map` is used to track visited URLs and prevent duplicate processing.
- **Graceful Shutdown:** The crawler respects context cancellation for clean shutdown.

### 3.3. `ContentCleaner`

The `ContentCleaner` uses the `goquery` library to parse and manipulate HTML. The cleaning process involves:

1. Removing common noise elements like `<nav>`, `<footer>`, `<script>`, and `<style>`.
2. Extracting the main content from `<article>` or `<main>` tags.
3. Using a simplified approach to convert the cleaned HTML to Markdown.

### 3.4. `QualityScorer`

The `QualityScorer` uses a weighted scoring algorithm based on:

- **Content Length:** A score is awarded for content between 200 and 5000 words.
- **Code Blocks:** The presence of code blocks increases the score.
- **Keyword Density:** The score is boosted by the presence of relevant keywords (e-g., "redis", "cluster").
- **Punctuation and Capitalization:** A small bonus is given for well-formatted content.

### 3.5. Configuration

The `crawler.yaml` file allows for flexible configuration of the crawler. See the file for a detailed example.

## 4. Future Improvements

- **JavaScript Rendering:** Use a headless browser to render JavaScript-heavy pages.
- **Advanced Markdown Conversion:** Use a more sophisticated HTML-to-Markdown library.
- **Machine Learning-based Classification:** Use a machine learning model for more accurate document classification.
