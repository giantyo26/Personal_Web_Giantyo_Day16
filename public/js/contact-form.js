// Contact form function
function getData(e) {
    e.preventDefault()

    // Get DOM value from form and initialized it to a variable
    let name = document.getElementById("name").value;
    let email = document.getElementById("email").value;
    let phone = document.getElementById("phone").value;
    let subject = document.getElementById("subject").value;
    let message = document.getElementById("message").value;

    if (name == "") {
        alert("Type out your name!")
    } else if (email == "") {
        alert("Input your email!")
    } else if (phone == "") {
        alert("Type your phone number!")
    } else if (subject == "") {
        alert("Choose the subject")
    } else if (message == "") {
        alert("Fill out the message")
    } else {
        const defaultEmail = "giantyoard@gmail.com";

        let mailTo = document.createElement("a");
        mailTo.href = `mailto:${defaultEmail}?subject=${subject}&body= Hello, my name is ${name}, ${message}, you can contact me on this number: ${phone}`;
        mailTo.click();

        // Form reset
        document.getElementById("contact-form").reset()
    }
}