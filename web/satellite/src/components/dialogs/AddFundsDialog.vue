// Copyright (C) 2025 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <v-dialog
        v-model="model"
        scrollable
        width="auto"
        min-width="400px"
        max-width="400px"
        transition="fade-transition"
        :persistent="isLoading"
    >
        <v-card rounded="xlg">
            <v-sheet>
                <v-card-item class="pa-6">
                    <template #prepend>
                        <div v-if="step === Step.Success" class="mr-2 rounded border pa-1">
                            <v-icon color="success" size="24" :icon="CircleCheckBig" />
                        </div>
                        <v-card-title class="font-weight-bold">
                            {{ step === Step.AddFunds ? 'Add Funds' : 'Payment Successful' }}
                        </v-card-title>
                    </template>

                    <template #append>
                        <v-btn
                            icon="$close"
                            variant="text"
                            size="small"
                            color="default"
                            @click="model = false"
                        />
                    </template>
                </v-card-item>
            </v-sheet>

            <v-divider />

            <v-card-text>
                <v-window v-model="step">
                    <v-window-item :value="Step.AddFunds">
                        <v-form v-model="formValid">
                            <p class="mb-7">Select your payment method:</p>

                            <v-select
                                v-model="selectedPaymentMethod"
                                label="Payment method"
                                :items="selectValues"
                                variant="outlined"
                                :rules="[RequiredRule]"
                                hide-details
                                class="mb-2"
                                required
                            >
                                <template #append-inner>
                                    <v-chip v-if="isDefaultSelected" color="default" size="small" variant="tonal" class="font-weight-bold">Default</v-chip>
                                </template>
                                <template #item="{ props, item }">
                                    <v-list-item v-bind="props">
                                        <template #append>
                                            <v-chip v-if="item.raw.isDefault" color="default" size="small" variant="tonal" class="font-weight-bold">Default</v-chip>
                                        </template>
                                    </v-list-item>
                                </template>
                            </v-select>

                            <p class="my-7">Choose the amount you wish to deposit:</p>

                            <v-text-field
                                v-model="amount"
                                label="Amount to deposit"
                                prefix="$"
                                type="number"
                                :min="minAmount"
                                :max="maxAmount"
                                :rules="amountRules"
                                hide-details="auto"
                                required
                                class="mb-2"
                                @update:model-value="onUpdateAmount"
                            />
                        </v-form>
                    </v-window-item>
                    <v-window-item :value="Step.Success">
                        <p>
                            Your payment of ${{ amount.toFixed(2) }} has been processed successfully.
                            Funds will be added to your account shortly.
                        </p>
                    </v-window-item>
                </v-window>
            </v-card-text>

            <v-divider />

            <v-card-actions class="pa-6">
                <v-row>
                    <v-col v-if="step === Step.AddFunds">
                        <v-btn variant="outlined" color="default" block @click="model = false">
                            Cancel
                        </v-btn>
                    </v-col>
                    <v-col>
                        <v-btn color="primary" variant="flat" block :loading="isLoading" :disabled="!formValid" @click="proceed">
                            {{ step === Step.AddFunds ? 'Continue' : 'Done' }}
                        </v-btn>
                    </v-col>
                </v-row>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import {
    VBtn,
    VCard,
    VCardActions,
    VCardItem,
    VCardText,
    VCardTitle,
    VChip,
    VCol,
    VDialog,
    VDivider,
    VForm,
    VIcon,
    VListItem,
    VRow,
    VSelect,
    VSheet,
    VTextField,
    VWindow,
    VWindowItem,
} from 'vuetify/components';
import { computed, ref, watch } from 'vue';
import { loadStripe } from '@stripe/stripe-js/pure';
import { Stripe } from '@stripe/stripe-js';
import { CircleCheckBig } from 'lucide-vue-next';

import { useLoading } from '@/composables/useLoading';
import { ChargeCardIntent, CreditCard } from '@/types/payments';
import { useBillingStore } from '@/store/modules/billingStore';
import { RequiredRule, ValidationRule } from '@/types/common';
import { useConfigStore } from '@/store/modules/configStore';
import { useNotify } from '@/composables/useNotify';
import { AnalyticsErrorEventSource } from '@/utils/constants/analyticsEventNames';

type SelectValue = {
    title: string;
    value: string;
    isDefault: boolean;
};

enum Step {
    AddFunds,
    Success,
}

const configStore = useConfigStore();
const billingStore = useBillingStore();

const notify = useNotify();
const { isLoading, withLoading } = useLoading();

const model = defineModel<boolean>({ required: true });

const step = ref<Step>(Step.AddFunds);
const formValid = ref<boolean>(false);
const selectedPaymentMethod = ref<string>();
const amount = ref<number>(10);
const isDefaultSelected = ref<boolean>(true);
const stripe = ref<Stripe | null>(null);

const amountRules = computed<ValidationRule<string>[]>(() => {
    return [
        RequiredRule,
        v => !(isNaN(+v) || isNaN(parseInt(v))) || 'Invalid number',
        v => !/[.,]/.test(v) || 'Value must be a whole number',
        v => (parseInt(v) > 0) || 'Value must be a positive number',
        v => {
            if (parseInt(v) > maxAmount.value) return `Amount must be less than or equal to ${maxAmount.value}`;
            if (parseInt(v) < minAmount.value) return `Amount must be more than or equal to ${minAmount.value}`;
            return true;
        },
    ];
});

const minAmount = computed<number>(() => configStore.state.config.minAddFundsAmount / 100);
const maxAmount = computed<number>(() => configStore.state.config.maxAddFundsAmount / 100);

const creditCards = computed((): CreditCard[] => billingStore.state.creditCards);

const selectValues = computed<SelectValue[]>(() => creditCards.value.map(card => {
    return { title: `${card.brand} **** ${card.last4}`, value: card.id, isDefault: card.isDefault };
}));

function onUpdateAmount(value: string): void {
    if (!value) {
        amount.value = 1;
        return;
    }

    const num = +value;
    if (isNaN(num) || isNaN(parseInt(value))) return;
    amount.value = num;
}

function proceed(): void {
    if (step.value === Step.Success) {
        model.value = false;
        return;
    }

    withLoading(async () => {
        if (!selectedPaymentMethod.value) return;

        try {
            const resp = await billingStore.addFunds(selectedPaymentMethod.value, amount.value * 100, ChargeCardIntent.AddFunds);
            if (resp.success) {
                notify.success('Payment confirmed! Your account balance will be updated shortly.');
                step.value = Step.Success;
            } else if (resp.paymentIntentID && resp.clientSecret) {
                await handlePaymentConfirmation(resp.clientSecret);
            } else {
                notify.error('Failed to add funds', AnalyticsErrorEventSource.ADD_FUNDS_DIALOG);
            }
        } catch (error) {
            notify.notifyError(error, AnalyticsErrorEventSource.ADD_FUNDS_DIALOG);
        }
    });
}

async function handlePaymentConfirmation(clientSecret: string) {
    try {
        if (!stripe.value) {
            stripe.value = await loadStripe(configStore.state.config.stripePublicKey);
        }

        if (!stripe.value) {
            notify.error('Stripe failed to initialize.', AnalyticsErrorEventSource.ADD_FUNDS_DIALOG);
            return;
        }

        const { error, paymentIntent } = await stripe.value.confirmCardPayment(
            clientSecret,
            {
                payment_method: selectedPaymentMethod.value,
            },
        );

        if (error) {
            notify.error(error.message ?? 'Payment confirmation failed.', AnalyticsErrorEventSource.ADD_FUNDS_DIALOG);
            return;
        }
        if (paymentIntent?.status !== 'succeeded') {
            notify.warning('Payment confirmation failed.');
        } else {
            notify.success('Payment confirmed! Your account balance will be updated shortly.');
            step.value = Step.Success;
        }
    } catch (error) {
        notify.error(error.message, AnalyticsErrorEventSource.ADD_FUNDS_DIALOG);
    }
}

watch(model, newVal => {
    if (!newVal) {
        if (step.value === Step.Success) {
            billingStore.getBalance().catch(() => {});
        }

        amount.value = 10;
        formValid.value = false;
        step.value = Step.AddFunds;
    }

    selectedPaymentMethod.value = creditCards.value.find(card => card.isDefault)?.id;
});

watch(selectedPaymentMethod, newVal => {
    isDefaultSelected.value = creditCards.value.some(
        card => card.isDefault && card.id === newVal,
    );
});
</script>
