# vemenichy!
A music server for my raspberry pi zero 2w. I just wanted a sexy frontend to play music on, and then things got out of hand. Making a headless server was merely a sidequest.

![Here's how the frontend UI looks](https://media3.giphy.com/media/v1.Y2lkPTc5MGI3NjExb3JjamJtY3J5enpqMWxpOW1ocXEyYzVtY3ZkbDltNWV1NzQxcDMzYyZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/weTaXH0Zd2d0M9PXTw/giphy.gif)

## Features
- Custom Web UI (main quest lol)
- A backend server on Go
  - Standard http server.
  - Interprocess manipulation of mpv to pause, skip etc.
  - Unified logging system for the frontend.
  - Search and download wrappers for yt-dlp.
  - Wrapper for mpv to play audio files.
- Pinggy global tunnelling
  - Creates a global deepweb link to host the server frontend on bootup.
  - Sends link to a dedicated Discord channel using webhooks.
- .service file for Debian environment to start server as soon as it boots.

## Hardware Requirements
- Raspberry Pi
  - I used a Zero 2W, should work on any model.
  - In fact, should work for any Linux machine.
  - Partially runs on windows - IPC commands fail.
- Power Source
  - Any power bank should suffice. Ideally 5V, 2.5A
  - May require MicroUSB (M) to Type C (F) OTG converter
- Audio device
  - MicroUSB (M) to Type C (F) OTG converter, 3.5 mm (F) to Type C (M) dongle required as per hardware constraints.
- MicroSD Card
  - MicroSD Card Reader :p

## Software Requirements
- Get a RasPi image on the SD Card.
  - Use the official imager for this. 
  - Raspberry Pi OS Lite (64-bit) is sufficient.
- Install the following:
  - Go 1.2x, update go.mod accordingly.
  - yt-dlp (latest version, ideally keep updating this regularly)
  - mpv
  - nodejs / bun / deno (Needed for solving JS puzzles for YouTube DRM.)

## External Services Required
- YouTube account (ideally adult.)
- Pinggy.io account (free tier sufficient.)
- Google Cloud Console Project API (for YouTube searches)
- Discord Account, Discord server with a webhook.

## Environment Variables
Have the following environment variables in the .env file in back/go_stuff:
```env
YOUTUBE_API_KEY=<from google cloud console>
webhookURL=<from Discord webhook>
sshToken=<visit pinggy.io, make a free account and grab a token, yes I know sshToken is a dumb name>
GLOBAL_PASSWORD=<This is a password for your frontend, it is asked if you request system commands through the web UI - enabling/disabling global tunnel or shutting the server down. Keep this password per your whims.>
```

## Installation and Setup
### Prerequisites
- SSH into your RasPi
- Install all the required software mentioned in the above section.
- Ensure that yt-dlp can see the path for nodejs, deno or bun.
### Cloning
- Clone this repository. In your desired directory, run the following command. Can remove '--depth 1' tag, but getting previous versions is redundant.
```bash 
git clone --depth 1 https://github.com/SupercalifragilisticexpialidociousPoppy/vemenichy.git
```
### Add sensitive information
- Make .env file in back/go_stuff
- YouTube requires your cookies whenever you try to download an age-restricted song to verify your age. This is why an adult YouTube account is required. Use browser extensions like 'Get cookies.txt LOCALLY' to obtain a cookies.txt file, put it in back/go_stuff.
  - Alternatively, if you only plan on downloading kid-friendly songs, you can choose not to get a cookies.txt file. You do need to modify the backend though otherwise you'll run into an error whenever you try to download. Go to /back/go_stuff/internal/api/handlers.go and consult lines 167 to 169.
### Adding background images
- You can add multiple image sets for background images (like for light and dark mode).
- Consult front/themes/white directory. Put your light mode images in this directory. Update carousel.json accordingly to store your current images.
- You can make additional image sets (like say for dark mode) by making a new folder (here, front/themes/dark), adding your images there, and creating a carousel.json file following the format given.
- Update index.html to have your modes reflect on the Settings menu buttons (my setup uses a light, dark and colours mode, you ought to change it according to your setup), update the javascript accrodingly as well.
- Credit the artists! Update index.html to include their names and socials.
### Creating a Binary File and Running
- Go to back/go_stuff, run the following go build command to create a binary executable named 'vemenichy-bin', and make it an executable. Note that you'll need to rebuild the binary every time you modify the code.
```bash
go build cmd/server/main.go -o vemenichy-bin
chmod +x vemenichy-bin
```
- Start the server by running the executable.
```bash
./vemenichy-bin
```
- Alternatively, you can use go run:
```bash
go run cmd/server/main.go
```
### Autoboot
Vemenichy server starts as soon as the RasPi boots up. This removes the need to ssh into RasPi to start the server every time.
- Copy/Move the vemenichy.service file in back/system to absolute path /etc/systemd/system.
- Run the following commands to make this .service file actually work from the next bootup.
```bash
sudo systemctl daemon-reload
sudo systemctl enable vemenichy.service
```
!! Ensure your filepaths are correct; the ones included in my code are for my environment; yours will be different.

## Usage
- For any device in the same local network as the Raspberry Pi, you can visit http://<name-of-your-raspberry-pi>.local:8080 or use the <explicit-IP-address>:8080 to open and run the server UI without the internet.
- The server also creates a global pinggy link, which can be used to access the UI outside of your local network as soon as it starts. You can comment this functionality out in back/go_stuff/cmd/server/main.go if you don't want it.
- Do note that your local network does require an internet connection to search and download songs. The local web UI doesn't use the internet, though.
