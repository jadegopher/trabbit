# Stu

Named after the alien from the short animation "Lifted," this tool helps test and develop trabbit.

It performs random actions to simulate activity in a RabbitMQ instance.

## Features

- **Concurrent Workers**: Run multiple workers in parallel using a worker pool.
- **Random Actions**: Workers randomly select and execute actions like Publish, Fetch, and Consume.
- **Concurrency Limits**: Actions can have global limits on how many instances run simultaneously (e.g., max 2 consumers).
- **Graceful Shutdown**: Responds to interrupt signals to clean up resources.

## Usage

```bash
go run ./cmd/stu -url "amqp://guest:guest@localhost:5672/" -queue "my-queue" -workers 5 -interval 1s
```

### Flags

- `-url`: RabbitMQ AMQP URL (default: "amqp://guest:guest@localhost:5672/").
- `-queue`: The queue to perform actions on (default: "stu-queue").
- `-workers`: Number of concurrent workers (default: 1).
- `-interval`: Time to wait between action cycles for each worker (default: 1s).

## Actions

- **Publish**: Sends a random text message to the queue.
- **Fetch**: Uses `basic.get` to poll a message (low overhead).
- **Consume**: Uses `basic.consume` to subscribe to the queue (appears as a "Consumer" in RabbitMQ). Limited to 2 parallel instances by default to allow message buildup.
- **Purge**: Clears all messages from the queue. Limited to 1 parallel instance.