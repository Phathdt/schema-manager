# Schema Manager - Project Overview

## Purpose
Schema Manager is a focused migration tool that compares Prisma schema files with existing migrations and generates missing migration files. It integrates seamlessly with Goose for migration execution, following the Unix philosophy of "do one thing and do it well."

## Key Features
- **Schema Diff**: Compare `schema.prisma` with existing migrations
- **Migration Generation**: Generate only missing migration files  
- **Goose Integration**: Let Goose handle migration execution
- **Clean Architecture**: Focused on core functionality
- **Database Introspection**: Import existing database structure into schema.prisma
- **Bi-directional Sync**: Sync between database and schema.prisma

## Tech Stack
- **Language**: Go 1.24.4
- **CLI Framework**: urfave/cli/v2 v2.27.7
- **Database Driver**: lib/pq v1.10.9 (PostgreSQL)
- **Build System**: Make
- **Migration Tool**: Goose (external integration)
- **Schema Format**: Prisma schema files

## Architecture
- Schema-first approach with Prisma schema as source of truth
- Goose-compatible migration generation
- Conditional SQL migrations for safe execution
- Clean CLI interface with focused commands

## Current Status
Version 0.1.3 with core features completed including introspect, sync, generate, and validate commands.