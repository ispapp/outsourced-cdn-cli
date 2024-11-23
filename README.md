## Overview

This project provides a backend for managing OpenWrt devices and uploading images to Cloudinary.

## Requirements

- Go 1.16+
- Cloudinary account
- .env file with `CLOUDINARY_URL` set

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/yourusername/ispapp_backend.git
    cd ispapp_backend
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
go run main.go serve --input path/to/cdn.db --port 8080
```

## Testing

To run the tests, use the following command:
```sh
go test ./...
```

This will run the tests defined in `main_test.go` to ensure the CLI commands are working correctly.
```