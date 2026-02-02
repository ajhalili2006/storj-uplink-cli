// Copyright (C) 2026 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <app-shell>
        <template #nav>
            <account-nav />
        </template>
    </app-shell>
</template>

<script setup lang="ts">
import AppShell from '@/layouts/shared/AppShell.vue';
import AccountNav from '@/layouts/account/AccountNav.vue';
</script>
