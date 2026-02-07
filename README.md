<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [lumeblog-cms](#lumeblog-cms)
  - [Structure](#structure)
  - [Usage](#usage)
    - [docker-compose (recommended)](#docker-compose-recommended)
    - [Local (without Docker)](#local-without-docker)
  - [Services and Ports](#services-and-ports)
  - [Persistent Data](#persistent-data)
  - [Initial Setup](#initial-setup)
  - [Public Assets Fallback](#public-assets-fallback)
  - [Notes](#notes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# lumeblog-cms

This is a PocketBase-backed blog app with a public site and a WYSIWYG admin CMS.

## Structure
- `backend/` PocketBase (Go) server.
- `frontend/` Vite + React public site and `/admin` CMS.

## Usage
### docker-compose (recommended)
1. `cd lumeblog-cms`
2. `docker-compose up --build` (builds and starts services)
3. Open PocketBase admin UI at `http://127.0.0.1:8090/_/`.
4. Complete initial setup (see "Initial Setup" below).

### Local (without Docker)
1. Start PocketBase:
   - `cd backend`
   - `go run .`
2. In another terminal, start the frontend:
   - `cd frontend`
   - `npm install`
   - `npm run dev`
3. If PocketBase is not on `http://127.0.0.1:8090`, set `VITE_PB_URL`.

## Services and Ports
- `8090`: PocketBase API + Admin UI (`/_/`)
- `5173`: Public site (Vite dev server)
- `5174`: Admin UI web app (`/admin` in the SPA build)

## Persistent Data
- Docker: stored in the named volume `pb_data` mounted at `/pb/pb_data`.
- Local (no Docker): stored in `lumeblog-cms/backend/pb_data`.

## Initial Setup
1. Open PocketBase admin UI at `http://127.0.0.1:8090/_/`.
2. Create the first PocketBase superuser (email + password).
3. Alternative (recommended for first launch): use the auto-generated one-time URL shown in the PocketBase logs on first boot.
   Example:
   ```
   2025/03/23 02:37:45 Server started at http://127.0.0.1:8090
   ├─ REST API:  http://127.0.0.1:8090/api/
   └─ Dashboard: http://127.0.0.1:8090/_/

   (!) Launch the URL below in the browser if it hasn't been open already to create your first superuser account:
   http://127.0.0.1:8090/_/#/pbinstal/<temporary-token>
   (you can also create your first superuser by running: ./pocketbase superuser upsert EMAIL PASS)
   ```
   Open the URL in a browser to access the superuser creation screen directly.
4. In the `cms_users` collection, create users with roles:
   - `admin`
   - `editor`
   - `viewer`
5. Log into the CMS at `http://127.0.0.1:5174` (or the deployed admin URL).
6. Ensure public content is published so it appears on the public site.

## Public Assets Fallback
- The SSR server serves static files from `frontend/public` by default.
- If `frontend/public` is empty, it automatically falls back to `frontend/default-public-asset`.
- Default assets live in `frontend/default-public-asset` (`styles.css`, `default-hero.svg`, `default-pattern.svg`).
- Add your own assets to `frontend/public` to override the defaults.

## Notes
- Public API exposure is controlled by PocketBase rules.
