<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [alleycat](#alleycat)
  - [Structure](#structure)
  - [Usage](#usage)
    - [docker-compose (recommended)](#docker-compose-recommended)
    - [Local (without Docker)](#local-without-docker)
  - [Services and Ports](#services-and-ports)
  - [Persistent Data](#persistent-data)
  - [Initial Setup](#initial-setup)
  - [Public Assets Fallback](#public-assets-fallback)
  - [Admin Settings](#admin-settings)
  - [Notes](#notes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# alleycat

This is a PocketBase-backed blog app with a public site and a WYSIWYG admin CMS.

## Structure
- `backend/` PocketBase (Go) server.
- `frontend/` Vite + React public site and `/admin` CMS.

## Usage
### docker-compose (recommended)
1. `cd alleycat`
2. `docker-compose up --build` (builds and starts services)
3. Open PocketBase admin UI at `http://127.0.0.1:8091/_/`.
4. Complete initial setup (see "Initial Setup" below).

### Local (without Docker)
1. Start PocketBase:
   - `cd backend`
   - `go run .`
2. In another terminal, start the frontend:
   - `cd frontend`
   - `npm install`
   - `npm run dev`
3. If PocketBase is not on `http://127.0.0.1:8091`, set `VITE_PB_URL`.

## Services and Ports
- `8091`: PocketBase API + Admin UI (`/_/`)
- `5173`: Public site (SSR server)
- `5175`: Admin UI web app

## Persistent Data
- Docker: stored in the named volume `pb_data` mounted at `/pb/pb_data`.
- Local (no Docker): stored in `alleycat/backend/pb_data`.

## Initial Setup
1. Open PocketBase admin UI at `http://127.0.0.1:8091/_/`.
2. Create the first PocketBase superuser (email + password).
3. Alternative (recommended for first launch): use the auto-generated one-time URL shown in the PocketBase logs on first boot.
   Example:
   ```
   2025/03/23 02:37:45 Server started at http://127.0.0.1:8091
   ├─ REST API:  http://127.0.0.1:8091/api/
   └─ Dashboard: http://127.0.0.1:8091/_/

   (!) Launch the URL below in the browser if it hasn't been open already to create your first superuser account:
   http://127.0.0.1:8091/_/#/pbinstal/<temporary-token>
   (you can also create your first superuser by running: ./pocketbase superuser upsert EMAIL PASS)
   ```
   Open the URL in a browser to access the superuser creation screen directly.
4. In the `cms_users` collection, create users with roles:
   - `admin`
   - `editor`
   - `viewer`
5. Log into the CMS at `http://127.0.0.1:5175` (or the deployed admin URL).
6. Ensure public content is published so it appears on the public site.

## Public Assets Fallback
- The SSR server serves static files from `frontend/public` by default.
- If `frontend/public` is empty, it automatically falls back to `frontend/default-public-asset`.
- Default assets live in `frontend/default-public-asset` (`styles.css`, `default-hero.svg`, `default-pattern.svg`).
- Add your own assets to `frontend/public` to override the defaults.

## Admin Settings
The following settings are editable in the Admin UI:
- Site name
- Description
- Welcome text
- Home top image
- Home top image alt
- Footer HTML
- Theme (Ember, Terminal, Wiki, Docs, Minimal). Disabled when `frontend/public` has assets.
- Site URL (feeds)
- Site language
- Enable post translation
- Translation source locale
- Translation target locales (multiple)
- Gemini model
- Gemini API key
- Feed items limit
- Enable RSS/Atom feed
- Enable JSON feed
- Enable code highlight
- Highlight theme
- Home page size
- Archive page size
- Show table of contents
- Show archive tags
- Show tags
- Show categories
- Show archive search slot
- Enable analytics
- Analytics URL
- Analytics site id
- Enable ads
- Ads client

### Post Translation Migration
- Existing posts can be translated with:
  - `cd backend`
  - `go run . translate-posts`
- This command reads translation options from `settings` and the Gemini API key from `app_secrets`.
- Gemini retry behavior is capped at 3 attempts per translation request.

### Backup Zip Import (CLI)
- You can import a PocketBase backup zip directly via command line:
  - `cd backend`
  - `go run . import-backup /path/to/pb_backup_xxx.zip`
- In Docker container:
  - `docker exec -it alleycat-pocketbase-1 /pb/pocketbase import-backup /pb/pb_data/backups/pb_backup_xxx.zip`
- The command replaces `pb_data` content from the specified zip (excluding `backups`, temp/cache internal dirs).
- Run this while the main PocketBase server process is stopped.

#### Docker Compose Import Steps
From the repository root (`alleycat`):
1. `docker compose stop pocketbase`
2. `docker compose run --rm -v "$PWD:/work" pocketbase /pb/pocketbase import-backup /work/pb_backup_xxx.zip`
3. `docker compose up -d pocketbase`

#### Important Backup Note
- PocketBase backup zip covers `pb_data` only.
- Custom frontend assets such as `frontend/public` CSS, images, and other static files are **not included** in the DB backup zip.
- Back up `frontend/public` separately (e.g. Git, tar/zip, or storage snapshot).

## Notes
- Public API exposure is controlled by PocketBase rules.
