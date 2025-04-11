<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink, RouterView } from 'vue-router'

//Function to set username cookie
const setCookie = (name: string, value: string) => {
    document.cookie = `${name}=${value}; path=/;`
}

//Function get a cookie by name
const getCookie = (name: string) => {
    const nameEQ = `${name}=`
    const ca = document.cookie.split(';')
    for (let i=0; i < ca.length; i++) {
        let c = ca[i]

        while (c.charAt(0) == ' ') c = c.substring(1, c.length)
        if (c.indexOf(nameEQ) == 0) return c.substring(nameEQ.length, c.length)
    }

    return null
}

const showModal = ref(false)
const username = ref('')

onMounted(() => {
    const cookie = getCookie('username')
    if (cookie) {
        username.value = cookie
    } else {
        showModal.value = true
    }
})

const submitUsername = () => {
    if (username.value.trim()) {
        setCookie('username', username.value)
        showModal.value = false
    } else {
        alert('Please enter a valid username')
    }
}
</script>


<template>
    <nav class="navbar">
        <div class="navbar-center">
            <h1 class="navbar-title">In spirit of R/place</h1>
        </div>
        <div class="navbar-actions">
            <button class="nav-button">{{ username || 'Username' }}</button>
            <button class="nav-button">Toggle Switch</button>
        </div>
    </nav>
    
    <div v-if="showModal" class="usernamemodal">
        <div class="modal-content">
            <h2>Enter your username</h2>
            <input v-model="username" type="text" placeholder="Username" />
            <button class="submit-button" @click="submitUsername">Submit</button>
        </div>
    </div>

    <RouterView />
</template>
