# potato
 
![template](https://user-images.githubusercontent.com/1962129/69706966-139a4580-10f9-11ea-82d3-3c752224cbd1.png)

**This project is experimental**, the purpose of this is to be able to manage services in multiple docker swarm node that are not joined as a cluster.

<img width="602" alt="Screen Shot 2019-11-27 at 8 46 34 AM" src="https://user-images.githubusercontent.com/1962129/69703769-7805d680-10f2-11ea-928e-166ce0a2f5d5.png">

So _potato_ will be able to communicate with each other, and manage docker swarm services on multiple nodes, just by simply changing the database value.

# Getting Started

Download [potato binaries from releases](https://github.com/alileza/potato/releases/latest).

## Starting on the server
```sh
// this will do default set-up, such as running database migration
// starting grpc server & http server for /metrics endpoint for prometheus
// "run potato -h" if you need some changes on configuration, such as port, etc.
potato -database-dsn "postgres://somewhere" server
```

**Notes:** postgres database set-up is required at the moment, on local development, you can simply use `docker-compose up -d postgres` and forget about the flag.

## Starting on the agent
```sh
potato -node-id [custom host id, by default it will be hostname] \
       -listen-address [wherever your potato server is] \
       agent
```

Once this set-up adding a new rows on database based on `node-id` would allow you to add/remove services from specific node you desire.
