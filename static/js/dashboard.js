// Dashboard JavaScript

let currentUser = null;

// Check authentication on page load
document.addEventListener('DOMContentLoaded', function() {
    const user = localStorage.getItem('user');
    if (!user) {
        // Not logged in, redirect to login page
        window.location.href = '/';
        return;
    }
    
    currentUser = JSON.parse(user);
    document.getElementById('username').textContent = currentUser.username;
    document.getElementById('userDisplayName').textContent = currentUser.username;
    
    // Load projects
    loadProjects();
});

// Load user's projects
async function loadProjects() {
    try {
        const response = await fetch('/api/projects', {
            headers: {
                'X-User-ID': currentUser.id
            }
        });
        
        if (response.ok) {
            const result = await response.json();
            displayProjects(result.projects);
        } else {
            console.error('Failed to load projects');
        }
    } catch (error) {
        console.error('Error loading projects:', error);
    }
}

// Display projects in the grid
function displayProjects(projects) {
    const projectsList = document.getElementById('projectsList');
    
    if (projects.length === 0) {
        projectsList.innerHTML = `
            <div class="empty-state" style="grid-column: 1 / -1; text-align: center; padding: 3rem;">
                <div style="font-size: 3rem; margin-bottom: 1rem;">📁</div>
                <h3>No projects yet</h3>
                <p>Create your first project to start collecting and analyzing logs.</p>
                <button onclick="showCreateProjectModal()" class="btn-primary" style="margin-top: 1rem;">
                    Create Your First Project
                </button>
            </div>
        `;
        return;
    }
    
    projectsList.innerHTML = projects.map(project => `
        <div class="project-card" onclick="openProject('${project.id}')">
            <h4>${project.name}</h4>
            <p>${project.searchable_keys.length} searchable keys</p>
            <div class="project-meta">
                <span>Created ${formatDate(project.created_at)}</span>
                <span>${project.ttl || 'No TTL'}</span>
            </div>
        </div>
    `).join('');
}

// Format date for display
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric' 
    });
}

// Show create project modal
function showCreateProjectModal() {
    document.getElementById('createProjectModal').style.display = 'block';
}

// Close create project modal
function closeCreateProjectModal() {
    document.getElementById('createProjectModal').style.display = 'none';
    document.getElementById('createProjectForm').reset();
}

// Create project form submission
document.getElementById('createProjectForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const formData = new FormData(this);
    const searchableKeys = formData.get('searchable_keys')
        ? formData.get('searchable_keys').split(',').map(key => key.trim()).filter(key => key)
        : [];
    
    const data = {
        name: formData.get('name'),
        searchable_keys: searchableKeys,
        ttl: formData.get('ttl') || null
    };
    
    try {
        const response = await fetch('/api/projects', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-User-ID': currentUser.id
            },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        if (response.ok) {
            alert('Project created successfully!');
            closeCreateProjectModal();
            loadProjects(); // Reload projects
        } else {
            alert(result.error || 'Failed to create project');
        }
    } catch (error) {
        console.error('Error creating project:', error);
        alert('An error occurred while creating the project');
    }
});

// Open project details page
function openProject(projectId) {
    window.location.href = `/project/${projectId}`;
}

// Logout function
function logout() {
    localStorage.removeItem('user');
    localStorage.removeItem('token');
    window.location.href = '/';
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('createProjectModal');
    if (event.target === modal) {
        closeCreateProjectModal();
    }
} 