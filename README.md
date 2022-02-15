# lexis-frontend-portal-backend-services

<a href="https://doi.org/10.5281/zenodo.6080537"><img src="https://zenodo.org/badge/DOI/10.5281/zenodo.6080537.svg" alt="DOI"></a>

This repo contains the WP8 Portal BackEnd for LEXIS.

## Acknowledgement
This code repository is a result / contains results of the LEXIS project. The project has received funding from the European Union’s Horizon 2020 Research and Innovation programme (2014-2020) under grant agreement No. 825532.

## Components

The golang application in this repository provides the following functionality:
- serve the compiled version of the React FE
- support an authentication flow
- serve information relating to the logged in user, including a token valid for services within the (keycloak) realm

The following endpoints are exposed:
- `/auth/login`
- `/auth/logout`
- `/auth/session-info`
- `/` - serves the react frontend

The repo contains
- the application in the `server` directory
- scripts to build the service in a docker container in the `build` directory
- support for running the service including configuration files in the `run` directory; note that it is recommended that these are copied elsewhere in the filesystem and run from there to minimize the likelihood of putting sensitive information (certs, secrets) into the code repo.


## Instructions for building the container

The process to build the docker image is a multi-stage building in docker containers.
To initiate the process perform the following:
```
cd build
./start.sh [-c CENTER] [-d]
```
By default the system is set to use the development branches, however that can be changed in the script.
Parameters:
- -c CENTER: Center needs to fit same spelling in the defaultCENTER.json you're going to use. Default behavior CENTER = LRZ.
- -d: Set's the environment to development, which makes the building use the local checked out code in both the FE and BE, so it will assume that ../wp8-portal exists on your work tree.

Default behaviour if options are no present is to use LRZ as center and use git as sources.

## Instructions for running the container locally

Copy the contents of the `run` subdirectory to somewhere else on the filesystem; for example in `$HOME/frontend-server`.

You will need to install a certificate in this directory, for example `$HOME/frontend-server/cert` and adapt the `docker-compose.yml` accordingly to mount and bind this directory.

Similarly it is also necessary to mount and bind a configuration file, an example configuration file has been provided in the `run` subdirectory - `config.toml`, this provides a good starting point for modifying the configuration of the system.

The service require interaction with keycloak, therefore requires a running instance of keycloak, instructions on how to run a keycloak instance locally can be found here;
 - Local Keycloak Server, https://www.keycloak.org/docs/latest/getting_started/index.html#securing-a-jboss-servlet-application
 - Keycloak Server Docker Image, https://hub.docker.com/r/jboss/keycloak/

You will then need to setup the appropriate Realms, Client etc... in keycloak.

Once keycloak is setup and running locally you will need to update the configuration file to control how the service interacts with keycloak, mainly set the host, port, realm, clientid, etc...

You also need to change the Domain in main.go:createSessionStore().

You should now be ready to run the service from the container, it is important to ensure that the correct ports are exposed in the `docker-compose.yml` file, both host and container.

Ensuring that you are in the same directory as the `docker-compose.yml` file and you have docker compose installed you can simply run the following;

```
docker-compose up
```

This will create a running instance of the service, to access this you can go to the following link in your web browser;

http://localhost:HOST_PORT

## Instructions for running the service locally

Make sure your back-end services are running and configure the server.
Run
```
go mod download
```

Checkout the wp8 front-end (https://code.it4i.cz/lexis/wp8/wp8-portal) and install the necessary files by running
```
cd <PATH>/wp8-portal/lexis-client/
npm run build
cp -r build/* <PATH>/gocode/src/code.it4i.cz/lexis/wp8/lexis-portal-be/server/build/
```

Run
```
cd <PATH>/gocode/src/code.it4i.cz/lexis/wp8/lexis-portal-be/server/build/
go run ../*.go
```
