<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
    <nav class="navbar">
        <div class="navbar-center">
            <h1 class="navbar-title">In spirit of R/place</h1>
        </div>
        <div class="navbar-actions">
            <button class="nav-button">Username</button>
            <button class="nav-button">Toggle Switch</button>
        </div>
    </nav>
    
    


    <RouterView />
</template>
