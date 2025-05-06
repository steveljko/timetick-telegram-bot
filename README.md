# Timetick Telegram Bot

## Introduction

This bot acts as a interface for the Timetick CLI application, allowing users to easily create and manage time tracking entries directly within Telegram. It features an API server that facilitates the seamless import of all tracking data back into the original CLI application.

## Features

- The application automatically starts both the Telegram bot and the API server without the need for any additional flags.
- Use the `gen-api-token` command to generate an API token, which secures the routes and protects access to the server. 
    ```bash
    > timetick-telegram-bot gen-api-token

    API Token generated successfully!
    I1lJgBLN5GFyG26HGy9J_M32aalQCC5S8XOsCB6sqr0=
    IMPORTANT: Save this token now. You won't be able to see it again!
    ```
