// Project Details JavaScript

let currentUser = null;
let currentProject = null;
let currentEvents = [];
let currentEventIndex = 0;
let currentEventDetails = [];
let selectedKeys = [];
let currentEventName = '';
let currentEventOffset = 0;
let currentEventLimit = 10;
let hasMoreEvents = false;
let currentEventTotal = 0;
let isLoadingMoreEvents = false;

// Check authentication and load project data
document.addEventListener('DOMContentLoaded', function() {
    console.log('Project page loaded');
    
    const user = localStorage.getItem('user');
    if (!user) {
        console.log('No user found, redirecting to login');
        window.location.href = '/';
        return;
    }
    
    try {
        currentUser = JSON.parse(user);
        console.log('Current user:', currentUser);
        document.getElementById('username').textContent = currentUser.username;
        
        // Get project ID from URL
        const projectId = window.location.pathname.split('/').pop();
        console.log('Project ID from URL:', projectId);
        
        if (projectId) {
            loadProjectDetails(projectId);
        } else {
            console.error('No project ID found in URL');
            alert('Invalid project URL');
            window.location.href = '/dashboard';
        }
    } catch (error) {
        console.error('Error parsing user data:', error);
        localStorage.removeItem('user');
        window.location.href = '/';
    }
});

// Load project details
async function loadProjectDetails(projectId) {
    console.log('Loading project details for:', projectId);
    
    try {
        const response = await fetch(`/api/projects/${projectId}`, {
            headers: {
                'X-User-ID': currentUser.id
            }
        });
        
        console.log('Project API response status:', response.status);
        
        if (response.ok) {
            currentProject = await response.json();
            console.log('Project data loaded:', currentProject);
            displayProjectDetails();
            loadEvents();
        } else {
            const errorData = await response.json();
            console.error('Project API error:', errorData);
            alert('Project not found or access denied: ' + (errorData.error || 'Unknown error'));
            window.location.href = '/dashboard';
        }
    } catch (error) {
        console.error('Error loading project:', error);
        alert('Error loading project details: ' + error.message);
    }
}

// Display project details
function displayProjectDetails() {
    console.log('Displaying project details');
    
    document.getElementById('projectName').textContent = currentProject.name;
    document.getElementById('projectTitle').textContent = currentProject.name;
    document.getElementById('projectId').textContent = currentProject.id;
    document.getElementById('projectCreated').textContent = formatDate(currentProject.created_at);
    document.getElementById('projectTtl').textContent = currentProject.ttl || 'No TTL';
    document.getElementById('apiKey').value = currentProject.api_key;
    
    // Display searchable keys
    const keysContainer = document.getElementById('searchableKeys');
    if (currentProject.searchable_keys && currentProject.searchable_keys.length > 0) {
        keysContainer.innerHTML = currentProject.searchable_keys.map(key => 
            `<span class="key-tag">${key}</span>`
        ).join('');
    } else {
        keysContainer.innerHTML = '<span>No searchable keys defined</span>';
    }
    
    // Create key filters
    const keyFiltersContainer = document.getElementById('keyFilters');
    if (currentProject.searchable_keys && currentProject.searchable_keys.length > 0) {
        keyFiltersContainer.innerHTML = currentProject.searchable_keys.map(key => `
            <div class="key-filter">
                <input type="checkbox" id="filter_${key}" value="${key}" onchange="updateFilters()">
                <label for="filter_${key}">${key}</label>
            </div>
        `).join('');
    } else {
        keyFiltersContainer.innerHTML = '<span>No keys available for filtering</span>';
    }
}

// Load events for the project
async function loadEvents() {
    console.log('Loading events for project:', currentProject.id);
    
    try {
        const response = await fetch(`/api/projects/${currentProject.id}/events`, {
            headers: {
                'X-User-ID': currentUser.id
            }
        });
        
        console.log('Events API response status:', response.status);
        
        if (response.ok) {
            const result = await response.json();
            console.log('Events data loaded:', result);
            currentEvents = result.events || [];
            displayEvents();
        } else {
            const errorData = await response.json();
            console.error('Events API error:', errorData);
            // Don't show error for events, just show empty state
            currentEvents = [];
            displayEvents();
        }
    } catch (error) {
        console.error('Error loading events:', error);
        currentEvents = [];
        displayEvents();
    }
}

// Display events in table
function displayEvents() {
    console.log('Displaying events, count:', currentEvents.length);
    
    const tbody = document.getElementById('eventsTableBody');
    
    if (currentEvents.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="4" style="text-align: center; padding: 3rem;">
                    <div style="font-size: 2rem; margin-bottom: 1rem;">📊</div>
                    <h3>No events yet</h3>
                    <p>Start sending events to this project to see them here.</p>
                    <div style="margin-top: 1rem; font-size: 0.875rem; color: #64748b;">
                        <p>Use the API key above to submit events:</p>
                        <pre style="text-align:left; background: #f1f5f9; padding: 1rem; border-radius: 0.5rem; overflow-x: auto;">
curl -X POST http://localhost:9090/api/logs \\
  -H "Content-Type: application/json" \\
  -d '{
    "project_id": "${currentProject.id}",
    "api_key": "${currentProject.api_key}",
    "payload": {
      "name": "test_event",
      "timestamp": "${new Date().toISOString()}",
      "data": {
        "user_id": "12345",
        "session_id": "67890"
      }
    }
  }'</pre>
                    </div>
                </td>
            </tr>
        `;
        return;
    }
    
    tbody.innerHTML = currentEvents.map(event => `
        <tr>
            <td class="event-name">${event.name}</td>
            <td class="event-timestamp">${formatDateTime(event.last_timestamp)}</td>
            <td class="event-count">${event.count}</td>
            <td>
                                        <button onclick="openEventDetails('${event.name}')" class="btn-secondary">View All Events</button>
            </td>
        </tr>
    `).join('');
}

// Update filters and reload events
function updateFilters() {
    selectedKeys = Array.from(document.querySelectorAll('.key-filter input:checked'))
        .map(checkbox => checkbox.value);
    
    console.log('Filters updated:', selectedKeys);
    loadFilteredEvents();
}

// Load events with current filters
async function loadFilteredEvents() {
    console.log('Loading filtered events with keys:', selectedKeys);
    
    try {
        const params = new URLSearchParams();
        if (selectedKeys.length > 0) {
            params.append('keys', selectedKeys.join(','));
        }
        
        const response = await fetch(`/api/projects/${currentProject.id}/events?${params}`, {
            headers: {
                'X-User-ID': currentUser.id
            }
        });
        
        if (response.ok) {
            const result = await response.json();
            currentEvents = result.events || [];
            displayEvents();
        }
    } catch (error) {
        console.error('Error loading filtered events:', error);
    }
}

// Clear all filters
function clearFilters() {
    document.querySelectorAll('.key-filter input').forEach(checkbox => {
        checkbox.checked = false;
    });
    selectedKeys = [];
    loadEvents();
}

// Open event details modal
async function openEventDetails(eventName) {
    console.log('Opening event details for:', eventName);

    try {
        currentEventIndex = 0;
        currentEventOffset = 0;
        currentEventName = eventName;
        hasMoreEvents = false;
        currentEventDetails = [];
        currentEventTotal = 0;

        const params = new URLSearchParams();
        params.append('name', eventName);
        params.append('limit', currentEventLimit.toString());
        params.append('offset', currentEventOffset.toString());
        if (selectedKeys.length > 0) {
            params.append('keys', selectedKeys.join(','));
        }

        const response = await fetch(`/api/projects/${currentProject.id}/events/details?${params}`, {
            headers: {
                'X-User-ID': currentUser.id
            }
        });

        if (response.ok) {
            const result = await response.json();
            currentEventDetails = result.events || [];
            currentEventTotal = result.total || 0;
            hasMoreEvents = currentEventDetails.length === currentEventLimit;

            displayEventDetails(currentEventDetails, eventName);
            document.getElementById('eventModal').style.display = 'block';
        } else {
            const errorData = await response.json();
            console.error('Event details API error:', errorData);
            alert('Error loading event details: ' + (errorData.error || 'Unknown error'));
        }
    } catch (error) {
        console.error('Error loading event details:', error);
        alert('Error loading event details: ' + error.message);
    }
}

// Display event details in modal
function displayEventDetails(events, eventName) {
    console.log('Displaying event details, events count:', events.length);
    
    if (events.length === 0) {
        document.getElementById('eventDetails').innerHTML = '<p>No events found with current filters.</p>';
        return;
    }
    
    const event = events[currentEventIndex];
    document.getElementById('eventModalTitle').textContent = `Event: ${eventName}`;
    document.getElementById('eventCounter').textContent = `${currentEventIndex + 1} of ${currentEventTotal}`;

    // Build data display HTML
    let dataHtml = '';
    if (event.data && Object.keys(event.data).length > 0) {
        dataHtml = `
            <div class="event-detail-item">
                <label>Event Data:</label>
                <div class="event-data">
                    ${Object.entries(event.data).map(([key, value]) => {
                        let valueStr = '';
                        if (value === null || value === undefined || value === '') {
                            valueStr = '<em>empty</em>';
                        } else if (value.length > 100) {
                            valueStr = `<pre>${value}</pre>`;
                        } else {
                            valueStr = value;
                        }
                        return `<div class="data-item">
                            <span class="data-key">${key}</span>
                            <span class="data-value">${valueStr}</span>
                        </div>`;
                    }).join('')}
                </div>
            </div>
        `;
    }
    
    const detailsHtml = `
        <div class="event-detail-item">
            <label>Event Name:</label>
            <span>${event.name}</span>
        </div>
        <div class="event-detail-item">
            <label>Timestamp:</label>
            <span>${formatDateTime(event.timestamp)}</span>
        </div>
        <div class="event-detail-item">
            <label>Created:</label>
            <span>${formatDateTime(event.created_at)}</span>
        </div>
        ${dataHtml}
    `;
    
    document.getElementById('eventDetails').innerHTML = detailsHtml;
}

// Navigate to previous event
function previousEvent() {
    if (currentEventIndex > 0) {
        currentEventIndex--;
        displayEventDetails(currentEventDetails, currentEventName);
    }
}

// Navigate to next event
async function nextEvent() {
    if (currentEventIndex < currentEventDetails.length - 1) {
        currentEventIndex++;
        displayEventDetails(currentEventDetails, currentEventName);
    } else if (hasMoreEvents) {
        // Load more events from ClickHouse
        await loadMoreEvents();
    } else {
        alert('No more events');
    }
}

// Load more events from ClickHouse
async function loadMoreEvents() {
    if (isLoadingMoreEvents) return;
    isLoadingMoreEvents = true;

    try {
        currentEventOffset += currentEventLimit;

        const params = new URLSearchParams();
        params.append('name', currentEventName);
        params.append('limit', currentEventLimit.toString());
        params.append('offset', currentEventOffset.toString());
        if (selectedKeys.length > 0) {
            params.append('keys', selectedKeys.join(','));
        }

        const response = await fetch(`/api/projects/${currentProject.id}/events/details?${params}`, {
            headers: {
                'X-User-ID': currentUser.id
            }
        });

        if (response.ok) {
            const result = await response.json();
            const newEvents = result.events || [];

            if (typeof result.total === 'number') {
                currentEventTotal = result.total;
            }

            if (newEvents.length > 0) {
                currentEventDetails = currentEventDetails.concat(newEvents);
                hasMoreEvents = newEvents.length === currentEventLimit;

                currentEventIndex = currentEventDetails.length - newEvents.length;
                displayEventDetails(currentEventDetails, currentEventName);
            } else {
                hasMoreEvents = false;
                alert('No more events');
            }
        } else {
            const errorData = await response.json();
            console.error('Error loading more events:', errorData);
            alert('Error loading more events: ' + (errorData.error || 'Unknown error'));
        }
    } catch (error) {
        console.error('Error loading more events:', error);
        alert('Error loading more events: ' + error.message);
    } finally {
        isLoadingMoreEvents = false;
    }
}

// Close event modal
function closeEventModal() {
    document.getElementById('eventModal').style.display = 'none';
}

// Copy API key to clipboard
function copyApiKey() {
    const apiKeyInput = document.getElementById('apiKey');
    apiKeyInput.select();
    document.execCommand('copy');
    alert('API key copied to clipboard!');
}

// Logout function
function logout() {
    localStorage.removeItem('user');
    localStorage.removeItem('token');
    window.location.href = '/';
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

// Format date and time for display
function formatDateTime(dateString) {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('eventModal');
    if (event.target === modal) {
        closeEventModal();
    }
} 