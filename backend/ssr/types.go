package main

type PBList[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

type PostRecord struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Body        string `json:"body"`
	Content     string `json:"content"`
	Excerpt     string `json:"excerpt"`
	Tags        string `json:"tags"`
	Category    string `json:"category"`
	Published   bool   `json:"published"`
	PublishedAt string `json:"published_at"`
	Date        string `json:"date"`
}

type PageRecord struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	URL         string `json:"url"`
	Body        string `json:"body"`
	Content     string `json:"content"`
	MenuVisible bool   `json:"menuVisible"`
	MenuOrder   int    `json:"menuOrder"`
	MenuTitle   string `json:"menuTitle"`
	Published   bool   `json:"published"`
	PublishedAt string `json:"published_at"`
	Date        string `json:"date"`
}

type SettingsRecord struct {
	ID                  string `json:"id"`
	SiteName            string `json:"site_name"`
	Description         string `json:"description"`
	WelcomeText         string `json:"welcome_text"`
	HomeTopImage        string `json:"home_top_image"`
	HomeTopImageAlt     string `json:"home_top_image_alt"`
	FooterHTML          string `json:"footer_html"`
	Theme               string `json:"theme"`
	SiteURL             string `json:"site_url"`
	SiteLanguage        string `json:"site_language"`
	EnableFeedXML       bool   `json:"enable_feed_xml"`
	EnableFeedJSON      bool   `json:"enable_feed_json"`
	FeedItemsLimit      int    `json:"feed_items_limit"`
	EnableAnalytics     bool   `json:"enable_analytics"`
	AnalyticsURL        string `json:"analytics_url"`
	AnalyticsSiteID     string `json:"analytics_site_id"`
	EnableAds           bool   `json:"enable_ads"`
	AdsClient           string `json:"ads_client"`
	EnableCodeHighlight bool   `json:"enable_code_highlight"`
	HighlightTheme      string `json:"highlight_theme"`
	ArchivePageSize     int    `json:"archive_page_size"`
	ExcerptLength       int    `json:"excerpt_length"`
	HomePageSize        int    `json:"home_page_size"`
	ShowToc             bool   `json:"show_toc"`
	ShowArchiveTags     bool   `json:"show_archive_tags"`
	ShowTags            bool   `json:"show_tags"`
	ShowCategories      bool   `json:"show_categories"`
	ShowArchiveSearch   bool   `json:"show_archive_search"`
}

type MediaRecord struct {
	ID      string `json:"id"`
	File    string `json:"file"`
	Caption string `json:"caption"`
	Path    string `json:"path"`
}
