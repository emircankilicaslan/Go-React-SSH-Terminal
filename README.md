#  Go-React-SSH-Terminal

A secure, full-stack web-based SSH terminal client built with **Go (Golang)** and **React**.
This project demonstrates a complete system architecture including JWT-based authentication, real-time WebSocket communication, and persistent server management via SQLite.

## 🏗️ Architecture Summary
The application follows a modern full-stack architecture:
* **Frontend:** Built with **React** and **Vite**. It uses `xterm.js` to render a real-time terminal interface in the browser.
* **Backend:** Developed with **Go (Golang)** using the `Gin` framework. It handles HTTP requests for authentication and manages WebSocket upgrades for SSH sessions.
* **Data Flow:**
    1.  User credentials are authenticated via a REST API (stored securely in **SQLite**).
    2.  SSH connection details are retrieved from the database.
    3.  A **WebSocket** connection is established between the frontend and backend.
    4.  The backend establishes a **real TCP SSH connection** to the remote server and pipes the `stdin/stdout` streams directly to the WebSocket, providing a live terminal experience.

## 🔌 How to Establish SSH Connection
1.  **Register/Login:** Create an account to access the dashboard.
2.  **Add Server:** Enter the **Host IP**, **Port** (usually 22), **Username**, and **Password** of the target Linux server.
3.  **Connect:** Click the **"Connect"** button next to the saved server.
    * The system validates the credentials.
    * A WebSocket channel opens.
    * The terminal interface appears, connected directly to the remote server.
  
 🤖 **AI Development Process Summary**
 
This project was developed with the strategic assistance of Artificial Intelligence (Claude 4.5 Opus & Gemini) as a technical consultant and pair programmer, while the overall architecture and logic were manually controlled and refined by the developer.

Architecture & Strategy: AI was used to evaluate the most efficient tech stack for the requirements. The decision to use Go for the backend and xterm.js for the terminal was finalized through these consultations.

Targeted Code Assistance: AI was utilized specifically for complex boilerplate sections, such as the initial setup of the WebSocket-SSH bridge and JWT middleware.

Optimization & Debugging: During development, AI helped in analyzing specific error logs and optimizing real-time data streaming between the backend and frontend.

Human-Led Development: All AI-generated suggestions were manually reviewed, debugged, and integrated into the final architecture by the developer to ensure system stability and security.



## 🛠️ Tech Stack
| Component | Technology |
|-----------|------------|
| **Backend** | Go (1.23), Gin, Gorilla WebSocket, GORM |
| **Frontend** | React.js, Vite, xterm.js |
| **Database** | SQLite (Embedded) |
| **Auth** | JWT & Bcrypt |

## 🚀 How to Run
1.  **Backend:** `cd backend` -> `go run main.go`
2.  **Frontend:** `cd frontend` -> `npm run dev`
3.  **Access:** Open `http://localhost:5173`

   <img width="533" height="760" alt="image" src="https://github.com/user-attachments/assets/097de53c-bf17-4b59-941b-3b961961b3ca" />

   <img width="1263" height="741" alt="image" src="https://github.com/user-attachments/assets/2fca6743-7d1b-4fb1-a33f-96e6d1464b4f" />


---
