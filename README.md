Cappa - Fast database snapshot and restore tool for development.
=======

Cappa allows you to quickly snapshot / revert database when you are e.g. writing database migrations, switching branches or messing with SQL. PostgreSQL only.

Heavily inspired by [fastmonkeys/stellar](https://github.com/fastmonkeys/stellar)

```
Cappa allows you to take fast snapshots / restore of your development database.
Useful when you have git branches containing migrations
Heavily (98%) inspired by fastmonkeys/stellar

Usage:
  cappa [command]

Available Commands:
  execute     Execute sql from file (default '.cappa.sql')
  help        Help about any command
  list        List all your snapshots
  remove      Remove snapshot
  restore     Restore from snapshot
  snapshot    Snapshot database
  version     Print the version number of Cappa

Flags:
  -h, --help      help for cappa
  -v, --verbose   What's wrong ? Speak to me

Use "cappa [command] --help" for more information about a command.


```

How it works
-------

Cappa works by storing copies of the database in the RDBMS (named as cappa_xxxx). 

When restoring the database, Cappa simply copy the database making it lot faster than the usual SQL dump. 

However, Cappa uses lots of storage space so you probably don't want to make too many snapshots or you will eventually run out of storage space.

**Warning: Please don't use Cappa if you can't afford data loss.** It's great for developing but not meant for production.

How to get started
-------

https://github.com/hbyio/cappa/releases

How to take a snapshot
-------

```$ cappa snapshot```

How to restore from a snapshot
-------

```$ cappa restore```

How to restore from a dump file
-------

```$ cappa restore dump```

If you load production data and need to run some sql before starting working (anonymisation)
-------

```$ cappa execute```

Common issues
-------

Make sure you have the rights to create new databases. 

If you are using PostreSQL, make sure you have a database named 'postgres'. 

Todo
-------

- [ ] UX (wait animations, warnings, ..)
- [ ] Tests
- [ ] Mysql

"Cappa" ?
-------
[Because it wants to take great snapshots](https://fr.wikipedia.org/wiki/Robert_Capa)