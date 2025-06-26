(function () {
    const toggleButton = document.getElementById('theme-toggle');
    const icon = document.getElementById('theme-toggle-icon');
    const root = document.documentElement;

    function setTheme(isDark) {
        if (isDark) {
            root.classList.add('dark');
            icon.textContent = 'light_mode';
            localStorage.setItem('theme', 'dark');
        } else {
            root.classList.remove('dark');
            icon.textContent = 'dark_mode';
            localStorage.setItem('theme', 'light');
        }
    }

    const saved = localStorage.getItem('theme');
    setTheme(saved === 'dark');

    if (toggleButton) {
        toggleButton.addEventListener('click', () => {
            const isDark = root.classList.contains('dark');
            setTheme(!isDark);
        });
    }
})();
