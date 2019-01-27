
Cargo v 1.0
------------

The cargo CLI use a "command" pattern, for example:

    cargo run ./src ./dst

This allows it to support new use cases more easily.

Cargo Run
---------

The Cargo run operation moves source files to the destination folder, processing the template files it encounters.

    cargo run ./src ./dst

Cargo Packaging
---------------

Create a simple repository management system to add/remove repositories, publish, index and download.

Add new commands:

    cargo repo add [repo-name] [ssh://|https://] --key ~/.ssh/id_rsa # add a repo + private key

    cargo repo refresh # re-index all the repositories - or a set of named [repo-name]

    cargo package cargo.yaml # create [cargo.name]-[cargo.version].tgz 

    cargo list # list all packages in all repositories
    
NOTE: the files are be uploaded outside of cargo.

Cargo Indexing
--------------

When a repo is added or refreshed, then all of the packages are downloaded, extracted and then cached.

The cache is stored at ~/./cargo/cache/[repo]/[package]/. 

This allows for quick installs and list/search operations.

The Cargo.yaml may contain additional meta data, such as CLI options:

    Cargo:
        keyFile: "./keys/id_rsa"

Cargo Manifest
--------------

A cargo package represents more than one set of templates.

Each package has a cargo.yaml file that describes it.

The Cargo key/value contains the "manifest" - the list of "from" folders that are packaged.

    Cargo:
        repo: troven # usually matches our repository name - not really used
        name: lab-demo-cargo # package name
        author:
            name: "Troven"
            email: "cto@troven.co"
        description: my first example package # free text
        version: 0.0.1 # mandatory semver
        partials: # preloaded partials/templates
            page:
                path: ./src/templates/page.html # no need for _
            product:
                path: ./src/templates/product.xml
        manifest: # maps names to folders
            abc:
                from: "./src/abc" # relative path in package
                to: "." # relative path at destination - prefixes output path
            def:
                from: "./src/def"
                to: "." # may overwrite existing files from ./src/abc/ 
            xyz:
                from: "./src/xyz"
                to: "./xyz" # preserve folder structure on output
                ignore: true

An empty "manifest" will default to:

    default:
        from " ./cargo"
        to: "."

The "to" field defaults to the name of the manifest "abc" or "def" if missing.

Cargo Install
-------------

When a package is installed, the files and folders references by manifest are rendered to the detination folder.

    cargo install [package-name] | [repo]/[package-name] ./dst

An install may generate multiple "run" operation per package. 

The --only require allows manifests to be whitelisted. This will only copy ./src/abc into to ./dst:

    cargo install troven/lab-demo-cargo --only abc  ./dst

When manfiests have overlapping destinations conflicting files - those last in the manifest take precedence.


It was the deliberate use of "." taht forced the overwrite behaviour in the example above.

ROADMAP
-------

1) Initil PoC v0.1.0
2) cargo run v0.2.0
3) Use Cargo.yaml if no context found - use as top-level Context v0.3.0
4) Cargo.partials - reference from template files v0.4.0
5) cargo install to install from Cargo.manifest  v0.5.0
6) cargo package - into tarball  v0.6.0
7) repo add - to download packages  v0.7.0
8) repo index / refresh / search  v0.8.0
9) feature freeze
10) training / documentation    v0.9.0
11) after UAT v1.0.0





