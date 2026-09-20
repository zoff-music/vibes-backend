# Zoff Backend Documentation

Start with the architecture for application and storage boundaries, then follow
the sequence diagrams for the request or background flow you are changing.

- [Architecture](ARCHITECTURE.md)
- [Application flows](FLOWS.md)
- [API contract](API.md)
- [Music providers](MUSIC-PROVIDERS.md)

The generated Swagger UI at `/api/swagger/` describes current route payloads.
PostgreSQL schema documentation belongs to the
[migrator](https://github.com/zoff-music/vibes-migrator/tree/main/docs/db);
runtime-specific setup belongs to the
[frontend apps](https://github.com/zoff-music/vibes-frontend#applications).
