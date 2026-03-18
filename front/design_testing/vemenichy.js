let fullscreen = false;
let stream_paused = false;
let typingsomething = false;
let track_num = 1;
const server_url = "http://Err-pi0.local:8080/";    

function fullscreen_pause_toggle() {
    const details = document.getElementById('full_scr_details');
    const pauseElement = document.getElementById('pause');

    if (stream_paused) {
        details.style.display = 'none';
        pauseElement.style.display = 'block'; // Shows the "PAUSED" text
    } else {
        details.style.display = 'block'; // Shows song info
        pauseElement.style.display = 'none';
    }
}

function toggle_fullscreen() {
    const fullScr = document.getElementById('full_scr');
    const fullScrDetails = document.getElementById('full_scr_details');
    const dashboard = document.querySelector('.dashboard');
    const header = document.getElementById('gimme_head');

    if (fullscreen) {
        // SHOW Fullscreen, HIDE Dashboard
        fullScr.style.display = "flex"; 
        dashboard.style.display = "none";
        header.style.display = "none";
        
        // Handle Details/Pause logic
        if (stream_paused) {
            fullScrDetails.style.display = "none";
            document.getElementById('pause').style.display = "block";
        } else {
            fullScrDetails.style.display = "block";
            document.getElementById('pause').style.display = "none";
        }
    } else {
        // HIDE Fullscreen, SHOW Dashboard
        fullScr.style.display = "none";
        dashboard.style.display = "grid";
        header.style.display = "block";
    }
}
async function send_cmd (endpoint) {
    try {
        const response = await fetch(`${server_url}${endpoint}`, {mode: 'cors'});
        if (response.ok) {
            console.log(`Command '${endpoint}' sent successfully.`);
        }
    } catch (error) {
        console.error(`Error sending command '${endpoint}':`, error);
    }
}

document.addEventListener('keydown', function(event) {
    if (event.key.toLowerCase() === 'k') {
        if (!typingsomething) {
            stream_paused = !stream_paused;
            if (fullscreen) {fullscreen_pause_toggle();}
        }
    }
});

document.addEventListener('keydown', function(event) {
    if (event.key.toLowerCase() === 'f') {
        if (!typingsomething) {
            fullscreen = !fullscreen;
            toggle_fullscreen();
        }
    }
});

send_cmd("/ping");