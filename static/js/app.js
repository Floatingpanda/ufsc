// App initialization
document.addEventListener('DOMContentLoaded', function() {
    console.log('UpForSchool app loaded');
    
    // Add any interactive features here
    const links = document.querySelectorAll('nav a');
    links.forEach(link => {
        if (link.href === window.location.href) {
            link.style.fontWeight = 'bold';
            link.style.color = '#3498db';
        }
    });
});
