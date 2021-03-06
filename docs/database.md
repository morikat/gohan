# Database

## SQL backend

Gohan generates SQL table definition based on a schema.

- string -> varchar(255)
- integer/number -> numeric
- boolean -> boolean

the other column will be "text".
We will encode data for JSON when we store complex data for RDBMS.


## YAML backend

YAML backend supports persistent data in YAML file format, and this backend is for only development purpose.

## Database Conversion tool

You can convert data source each other using convert tool.

```
  NAME:
     convert - Convert DB

  USAGE:
     command convert [command options] [arguments...]

  DESCRIPTION:
     Gohan convert can be used to migrate Gohan resources between different types of databases

  OPTIONS:
     --in-type, --it      Input db type (yaml, json, sqlite3)
     --in, -i             Input db connection spec (or filename)
     --out-type, --ot     Output db type (yaml, json, sqlite3)
     --out, -o            Output db connection spec (or filename)
     --schema, -s         Schema file
```

## Database Migration

Gohan supports generating goose (https://bitbucket.org/liamstask/goose) migration script.
Currently, Gohan doesn't provide function calculating diff on a schema, so that app developers should manage this migration script.

```

  NAME:
     migrate - Generate goose migration script

  USAGE:
     command migrate [command options] [arguments...]

  DESCRIPTION:
     Generates goose migration script

  OPTIONS:
     --name, -n 'init_schema'        name of migrate
     --schema, -s             Schema definition
     --path, -p 'etc/db/migrations'    Migrate path
     --cascade                If true, FOREIGN KEYS in database will be created with ON DELETE CASCADE
```

