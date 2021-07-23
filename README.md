This is a brutally butchered version of Joyent's [pg_prefaulter](https://github.com/joyent/pg_prefaulter).

Metrics removed, go modules added and some minor edits to the code to pass _staticcheck_ linting.

Fixed an issue where in pg10+, the code would attempt to prefault files just ahead of the WAL files most recently _received_, instead of files just ahead of latest WAL files most recently _replayed_.

To build,

    go build -o pg_prefaulter main.go

Please see the [original README](https://github.com/joyent/pg_prefaulter/blob/master/README.adoc) for motivation and further usage instructions.
