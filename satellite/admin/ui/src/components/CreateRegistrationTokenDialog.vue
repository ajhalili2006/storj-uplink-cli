// Copyright (C) 2026 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <RequireReasonFormDialog
        v-if="!createdToken"
        v-model="model"
        :loading="isLoading"
        :initial-form-data="initialFormData"
        :form-config="formConfig"
        title="Create Registration Token"
        subtitle="Create a token that allows a user to register"
        width="500"
        @submit="onSubmit"
    />

    <v-dialog v-else v-model="model" transition="fade-transition" width="500">
        <v-card rounded="xlg">
            <v-card-item class="pa-6">
                <template #prepend>
                    <v-card-title class="font-weight-bold">
                        Registration Token Created
                    </v-card-title>
                </template>
                <template #append>
                    <v-btn
                        :icon="X"
                        variant="text"
                        size="small"
                        color="default"
                        @click="closeAndReset"
                    />
                </template>
            </v-card-item>

            <v-divider />

            <div class="pa-6">
                <v-alert type="success" variant="tonal" class="mb-4">
                    Token created successfully! Share this token or registration link with the user.
                </v-alert>
                <TextOutputArea label="Registration Token" :value="createdToken" class="mb-4" />
                <TextOutputArea label="Registration Link" :value="registrationLink" />
            </div>

            <v-divider />

            <v-card-actions class="pa-6">
                <v-btn
                    color="primary"
                    variant="flat"
                    block
                    @click="closeAndReset"
                >
                    Done
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import {
    VDialog,
    VCard,
    VCardItem,
    VCardTitle,
    VCardActions,
    VBtn,
    VDivider,
    VAlert,
} from 'vuetify/components';
import { X } from 'lucide-vue-next';

import { useUsersStore } from '@/store/users';
import { useLoading } from '@/composables/useLoading';
import { useNotify } from '@/composables/useNotify';
import { FieldType, FormConfig } from '@/types/forms';
import { PositiveNumberRule, RequiredRule } from '@/types/common';
import { useAppStore } from '@/store/app';

import RequireReasonFormDialog from '@/components/RequireReasonFormDialog.vue';
import TextOutputArea from '@/components/TextOutputArea.vue';

const usersStore = useUsersStore();
const appStore = useAppStore();

const model = defineModel<boolean>({ required: true });

const { isLoading, withLoading } = useLoading();
const notify = useNotify();

const createdToken = ref<string | null>(null);

const initialFormData = computed(() => ({ projectLimit: null }));

const formConfig = computed((): FormConfig => {
    return {
        sections: [
            {
                rows: [
                    {
                        fields: [
                            {
                                key: 'projectLimit',
                                type: FieldType.Number,
                                label: 'Project Limit',
                                min: 1,
                                step: 1,
                                rules: [RequiredRule, PositiveNumberRule],
                                required: true,
                            },
                        ],
                    },
                ],
            },
        ],
    };
});

const registrationLink = computed<string>(() => {
    if (!createdToken.value) return '';

    const url = new URL('/signup', appStore.state.settings.console.externalAddress);
    url.searchParams.set('token', createdToken.value);

    return url.toString();
});

function closeAndReset(): void {
    model.value = false;
}

function onSubmit(formData: Record<string, unknown>): void {
    withLoading(async () => {
        try {
            createdToken.value = await usersStore.createRegistrationToken(
                formData.projectLimit as number,
                formData.reason as string,
            );
            notify.success('Registration token created successfully!');
        } catch (error) {
            notify.error(`Failed to create token: ${error.message}`);
        }
    });
}

watch(model, (newValue) => {
    if (!newValue) {
        // clear token after dialog close animation
        setTimeout(() => createdToken.value = null, 300);
    }
});
</script>
