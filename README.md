## Overview

This project provides a backend for managing devices illustrations and uploading images to Cloudinary.

## Requirements

- Go 1.16+
- Cloudinary account
- .env file with `CLOUDINARY_URL` set

## Installation

1. Clone the repository:
    ```sh
    git clone git@github.com:ispapp/outsourced-cdn-cli.git
    cd outsourced-cdn-cli
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

3. Create a `.env` file with your Cloudinary URL:
    ```sh
    echo "CLOUDINARY_URL=cloudinary://<api_key>:<api_secret>@<cloud_name>" > .env
    ```

## Usage

### Upload Images

To upload images to Cloudinary, use the `upload` command:
```sh
go run main.go upload --input path/to/input.json --folder cloudinary-folder --output path/to/cdn.db --env
```

### Serve Images

To serve images and proxy output JSON with fast search, use the `serve` command:
```sh
go run main.go serve --input path/to/cdn.db 
```

in case of cloudflared tunnel setup you need to follow the steps below:
-  install cloudflared
-  check the official documentation for the setup [link](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/get-started/create-local-tunnel/).
- save the config.yml file in the root directory of the project
- run the following command:
```sh
go run main.go serve --input path/to/cdn.db --cfd-config config.yml
```

## Testing

To run the tests, use the following command:
```sh
go test -v
```

This will run the tests defined in `main_test.go` to ensure the CLI commands are working correctly.
```
