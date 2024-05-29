
# Air Quality Data Parser

This project scrapes air quality data from `earth.nullschool.net` and stores it in a PostgreSQL database. It uses `chromedp` to fetch data from the website and a Docker setup to run the application and database in containers.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Prerequisites

- [Docker](https://www.docker.com/get-started)

### Steps

1. **Clone the repository:**

   ```bash
   git clone https://github.com/verysaddrug/air-quality-parser.git
   cd air-quality-parser
   ```

2. **Build and run the Docker containers:**

   ```bash
   docker-compose up --build
   ```

   This command will build the Docker images and start the containers for the application and PostgreSQL database.

## Usage

### Running the Scraper

To run the parser, simply start the Docker containers as shown above. The application will connect to `earth.nullschool.net`, fetch air quality data (PM1, PM2.5, PM10), and store it in the PostgreSQL database.

You can configure the latitude, longitude, start date, and end date for data scraping by modifying the command in `docker-compose.yml` or by passing them as arguments:

```bash
docker-compose run app --lat 60.1695 --lon 24.9354 --start 2023-01-01 --end 2024-01-01
```

### Accessing the Database

To access the PostgreSQL database, you can use any PostgreSQL client. The default credentials are specified in the `docker-compose.yml` file:

- Host: `localhost`
- Port: `5432`
- User: `postgres`
- Password: `qwerty`
- Database: `postgres`

Example connection string:

```plaintext
postgresql://postgres:qwerty@localhost:5432/postgres
```

## Configuration

### Environment Variables

You can configure the database connection using environment variables defined in the `.env` file:

```
DB_HOST=db
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=qwerty
DB_NAME=postgres
DB_SSLMODE=disable
```

### Command-Line Arguments

- `--lat`: Latitude of the location (default: 60.1695)
- `--lon`: Longitude of the location (default: 24.9354)
- `--start`: Start date in YYYY-MM-DD format (default: 2023-01-01)
- `--end`: End date in YYYY-MM-DD format (default: 2024-01-01)

## Contributing

We welcome contributions! Please follow these steps to contribute:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-branch`).
3. Make your changes.
4. Commit your changes (`git commit -m 'Add some feature'`).
5. Push to the branch (`git push origin feature-branch`).
6. Create a new Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
