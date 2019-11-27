create table releases (
    release_id serial primary key,
    hostname varchar(64) NOT NULL,
    version varchar(64) NOT NULL,
    ports varchar(64),
    replicas int NOT NULL default 1,
    created_by varchar(64) NOT NULL,
    created_at timestamp default current_timestamp
);