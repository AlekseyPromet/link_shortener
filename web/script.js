document.getElementById('firstForm').addEventListener('submit', async function (e) {
    e.preventDefault(); // Prevent default form submission

    const formData = new FormData(this);

    try {
        const response = await fetch('/short', {
            method: 'POST',
            body: formData,
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            }
        });

        if (!response.ok) {
            throw new Error('Network response was not ok');
        }

        const text = await response.text();
        console.log(text);

        // Display the response in the result div
        document.getElementById('result').textContent = text;
    } catch (error) {
        console.error('There was a problem with the fetch operation:', error.message);
        document.getElementById('result').textContent = 'Error submitting form: ' + error.message;
    }
});
