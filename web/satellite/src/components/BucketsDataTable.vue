// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

<template>
    <v-text-field
        v-model="search"
        label="Search"
        :prepend-inner-icon="Search"
        single-line
        variant="solo-filled"
        flat
        hide-details
        clearable
        density="comfortable"
        xl11 :maxlength="MAX_SEARCH_VALUE_LENGTH"
        class="mb-5"
    />

    <v-data-table-server
        :sort-by="sortBy"
        :headers="headers"
        :items="displayedItems"
        :search="search"
        :loading="areBucketsFetching"
        :items-length="page.totalCount"
        items-per-page-text="Buckets per page"
        :items-per-page-options="tableSizeOptions(page.totalCount)"
        no-data-text="No buckets found"
        class="border"
        hover
        @update:items-per-page="onUpdateLimit"
        @update:page="onUpdatePage"
        @update:sort-by="onUpdateSort"
    >
        <template #item.name="{ item }">
            <v-btn
                class="rounded-lg w-100 pl-1 pr-3 ml-n1 justify-start"
                variant="text"
                height="40"
                color="default"
                :disabled="bucketsBeingDeleted.has(item.name)"
                @click="openBucket(item.name)"
            >
                <template #default>
                    <img class="mr-3" src="@/assets/icon-bucket-tonal.svg" alt="Bucket">
                    <div class="max-width">
                        <p class="font-weight-bold text-lowercase text-truncate">{{ item.name }}</p>
                    </div>
                </template>
            </v-btn>
        </template>
        <template #item.storage="{ item }">
            <span>
                {{ Size.toBase10String(item.storage * Memory.GB) }}
            </span>
        </template>
        <template #item.egress="{ item }">
            <span>
                {{ item.egress.toFixed(2) + 'GB' }}
            </span>
        </template>
        <template #item.objectCount="{ item }">
            <span>
                {{ item.objectCount.toLocaleString() }}
            </span>
        </template>
        <template #item.segmentCount="{ item }">
            <span>
                {{ item.segmentCount.toLocaleString() }}
            </span>
        </template>
        <template #item.since="{ item }">
            <span class="text-no-wrap">
                {{ Time.formattedDate(item.createdAt) }}
            </span>
        </template>
        <template #item.location="{ item }">
            <div class="text-no-wrap">
                <v-icon size="28" class="mr-1 pa-1 rounded-lg border">
                    <component :is="item.location === 'global' ? Earth : LandPlot" :size="18" />
                </v-icon>
                <v-chip variant="tonal" :color="item.location === 'global' ? 'success' : 'primary'" size="small" class="text-capitalize">
                    {{ item.location || `unknown(${item.defaultPlacement})` }}
                </v-chip>
            </div>
        </template>
        <template #item.versioning="{ item }">
            <div class="text-no-wrap">
                <v-tooltip location="top" :text="getVersioningInfo(item.versioning)">
                    <template #activator="{ props }">
                        <v-icon v-bind="props" size="28" :icon="getVersioningIcon(item.versioning)" class="mr-1 pa-1 rounded-lg border" />
                    </template>
                </v-tooltip>
                <v-chip variant="tonal" :color="getVersioningChipColor(item.versioning)" size="small">
                    {{ item.versioning }}
                </v-chip>
            </div>
        </template>
        <template #item.objectLockEnabled="{ item }">
            <div class="text-no-wrap">
                <v-tooltip location="top" :text="getObjectLockInfo(item)">
                    <template #activator="{ props }">
                        <v-icon v-bind="props" size="28" :icon="item.objectLockEnabled ? LockKeyhole : LockKeyholeOpen" class="mr-1 pa-1 rounded-lg border" />
                    </template>
                </v-tooltip>
                <v-chip variant="tonal" :color="item.objectLockEnabled ? 'success' : 'default'" size="small">
                    {{ item.objectLockEnabled ? 'Enabled' : 'Disabled' }}
                </v-chip>
            </div>
        </template>
        <template #item.actions="{ item }">
            <v-tooltip v-if="bucketsBeingDeleted.has(item.name)" location="top" text="Deleting bucket">
                <template #activator="{ props }">
                    <v-progress-circular width="2" size="22" color="error" indeterminate v-bind="props" />
                </template>
            </v-tooltip>
            <v-menu v-else location="bottom end" transition="scale-transition">
                <template #activator="{ props: activatorProps }">
                    <v-btn
                        title="Bucket Actions"
                        :icon="Ellipsis"
                        color="default"
                        variant="outlined"
                        size="small"
                        rounded="md"
                        density="comfortable"
                        v-bind="activatorProps"
                    />
                </template>
                <v-list class="pa-1">
                    <v-list-item
                        density="comfortable"
                        link
                        @click="openBucket(item.name)"
                    >
                        <template #prepend>
                            <component :is="ArrowRight" :size="18" />
                        </template>
                        <v-list-item-title
                            class="ml-3 text-body-2 font-weight-medium"
                        >
                            Open Bucket
                        </v-list-item-title>
                    </v-list-item>
                    <div>
                        <v-list-item
                            v-if="versioningUIEnabled && item.versioning !== Versioning.NotSupported"
                            density="comfortable"
                            link
                            :disabled="item.versioning === Versioning.Enabled && item.objectLockEnabled"
                            @click="() => onToggleVersioning(item)"
                        >
                            <template #prepend>
                                <component :is="History" v-if="item.versioning !== Versioning.Enabled" :size="18" />
                                <component :is="CirclePause" v-else :size="18" />
                            </template>
                            <v-list-item-title class="ml-3">
                                {{ item.versioning !== Versioning.Enabled ? 'Enable Versioning' : 'Suspend Versioning' }}
                            </v-list-item-title>
                        </v-list-item>
                        <v-tooltip
                            v-if="item.versioning === Versioning.Enabled && item.objectLockEnabled"
                            activator="parent"
                            location="left"
                            max-width="300"
                        >
                            Versioning cannot be suspended on a bucket with object lock enabled
                        </v-tooltip>
                    </div>
                    <v-list-item v-if="showLockActionItem(item)" link @click="() => showSetBucketObjectLockDialog(item.name)">
                        <template #prepend>
                            <component :is="Lock" :size="18" />
                        </template>
                        <v-list-item-title class="ml-3">
                            {{ item.objectLockEnabled ? 'Lock Settings' : 'Enable Lock' }}
                        </v-list-item-title>
                    </v-list-item>
                    <v-list-item link @click="() => showShareBucketDialog(item.name)">
                        <template #prepend>
                            <component :is="Share" :size="18" />
                        </template>
                        <v-list-item-title class="ml-3">
                            Share Bucket
                        </v-list-item-title>
                    </v-list-item>
                    <v-list-item v-if="downloadPrefixEnabled" link @click="() => onDownloadBucket(item.name)">
                        <template #prepend>
                            <component :is="DownloadIcon" :size="18" />
                        </template>
                        <v-list-item-title class="ml-3">
                            Download Bucket
                        </v-list-item-title>
                    </v-list-item>
                    <v-list-item link @click="() => showBucketDetailsModal(item.name)">
                        <template #prepend>
                            <component :is="ReceiptText" :size="18" />
                        </template>
                        <v-list-item-title class="ml-3">
                            Bucket Details
                        </v-list-item-title>
                    </v-list-item>
                    <v-divider class="my-1" />
                    <v-list-item class="text-error text-body-2" link @click="() => showDeleteBucketDialog(item.name)">
                        <template #prepend>
                            <component :is="Trash2" :size="18" />
                        </template>
                        <v-list-item-title class="ml-3">
                            Delete Bucket
                        </v-list-item-title>
                    </v-list-item>
                </v-list>
            </v-menu>
        </template>
    </v-data-table-server>
    <delete-bucket-dialog v-model="isDeleteBucketDialogShown" :bucket-name="bucketToDelete" />
    <enter-bucket-passphrase-dialog v-model="isBucketPassphraseDialogOpen" @passphrase-entered="passphraseDialogCallback" />
    <share-dialog v-model="isShareBucketDialogShown" :bucket-name="shareBucketName" />
    <bucket-details-dialog v-model="isBucketDetailsDialogShown" :bucket-name="bucketDetailsName" />
    <set-bucket-object-lock-config-dialog v-if="objectLockUIEnabled" v-model="isSetBucketObjectLockDialogShown" :bucket-name="bucketObjectLockName" />
    <toggle-versioning-dialog v-model="bucketToToggleVersioning" @toggle="fetchBuckets" />
    <download-prefix-dialog v-if="downloadPrefixEnabled" v-model="isDownloadPrefixDialogShown" :prefix-type="DownloadPrefixType.Bucket" :bucket="bucketToDownload" />
</template>

<script setup lang="ts">
import { computed, FunctionalComponent, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import {
    VBtn,
    VChip,
    VDataTableServer,
    VDivider,
    VIcon,
    VList,
    VListItem,
    VListItemTitle,
    VMenu,
    VProgressCircular,
    VTextField,
    VTooltip,
} from 'vuetify/components';
import {
    ArrowRight,
    CircleCheck,
    CircleHelp,
    CircleMinus,
    CirclePause,
    CircleX,
    DownloadIcon,
    Earth,
    Ellipsis,
    History,
    LandPlot,
    Lock,
    LockKeyhole,
    LockKeyholeOpen,
    ReceiptText,
    Search,
    Share,
    Trash2,
} from 'lucide-vue-next';

import { Memory, Size } from '@/utils/bytesSize';
import { Bucket, BucketCursor, BucketMetadata, BucketPage } from '@/types/buckets';
import { useBucketsStore } from '@/store/modules/bucketsStore';
import { useConfigStore } from '@/store/modules/configStore';
import { useNotify } from '@/composables/useNotify';
import { AnalyticsErrorEventSource } from '@/utils/constants/analyticsEventNames';
import { useProjectsStore } from '@/store/modules/projectsStore';
import { DEFAULT_PAGE_LIMIT } from '@/types/pagination';
import { DataTableHeader, MAX_SEARCH_VALUE_LENGTH, tableSizeOptions } from '@/types/common';
import { EdgeCredentials } from '@/types/accessGrants';
import { ROUTES } from '@/router';
import { usePreCheck } from '@/composables/usePreCheck';
import { Versioning } from '@/types/versioning';
import { Time } from '@/utils/time';
import { useObjectBrowserStore } from '@/store/modules/objectBrowserStore';
import { capitalizedMode, NO_MODE_SET } from '@/types/objectLock';
import { DownloadPrefixType } from '@/types/browser';

import DeleteBucketDialog from '@/components/dialogs/DeleteBucketDialog.vue';
import EnterBucketPassphraseDialog from '@/components/dialogs/EnterBucketPassphraseDialog.vue';
import ShareDialog from '@/components/dialogs/ShareDialog.vue';
import BucketDetailsDialog from '@/components/dialogs/BucketDetailsDialog.vue';
import ToggleVersioningDialog from '@/components/dialogs/ToggleVersioningDialog.vue';
import SetBucketObjectLockConfigDialog from '@/components/dialogs/SetBucketObjectLockConfigDialog.vue';
import DownloadPrefixDialog from '@/components/dialogs/DownloadPrefixDialog.vue';

const bucketsStore = useBucketsStore();
const obStore = useObjectBrowserStore();
const projectsStore = useProjectsStore();
const configStore = useConfigStore();

const notify = useNotify();
const router = useRouter();
const { withTrialCheck, withManagedPassphraseCheck } = usePreCheck();

const FIRST_PAGE = 1;
const areBucketsFetching = ref<boolean>(true);
const search = ref<string>('');
const searchTimer = ref<NodeJS.Timeout>();
const bucketDetailsName = ref<string>('');
const bucketObjectLockName = ref<string>('');
const shareBucketName = ref<string>('');
const isDeleteBucketDialogShown = ref<boolean>(false);
const bucketToDelete = ref<string>('');
const isBucketPassphraseDialogOpen = ref(false);
const isShareBucketDialogShown = ref<boolean>(false);
const isSetBucketObjectLockDialogShown = ref<boolean>(false);
const isBucketDetailsDialogShown = ref<boolean>(false);
const isDownloadPrefixDialogShown = ref<boolean>(false);
const bucketToDownload = ref<string>('');
const pageWidth = ref<number>(document.body.clientWidth);
const sortBy = ref<SortItem[] | undefined>([{ key: 'name', order: 'asc' }]);
const bucketToToggleVersioning = ref<BucketMetadata | null>(null);

let passphraseDialogCallback: () => void = () => {};

type SortItem = {
    key: keyof Bucket;
    order: boolean | 'asc' | 'desc';
};

const displayedItems = computed<Bucket[]>(() => {
    const items = page.value.buckets;

    sort(items, sortBy.value);

    return items;
});

const downloadPrefixEnabled = computed<boolean>(() => configStore.state.config.downloadPrefixEnabled);

const showRegionTag = computed<boolean>(() => {
    return configStore.state.config.enableRegionTag;
});

/**
 * Whether versioning is enabled for current project.
 */
const versioningUIEnabled = computed(() => configStore.state.config.versioningUIEnabled);

/**
 * Whether object lock is enabled for current project.
 */
const objectLockUIEnabled = computed<boolean>(() => configStore.state.config.objectLockUIEnabled);

const isTableSortable = computed<boolean>(() => {
    return page.value.totalCount <= cursor.value.limit;
});

const headers = computed<DataTableHeader[]>(() => {
    const hdrs: DataTableHeader[] = [
        {
            title: 'Bucket',
            align: 'start',
            key: 'name',
            sortable: isTableSortable.value,
        },
        { title: 'Objects', key: 'objectCount', sortable: isTableSortable.value },
        { title: 'Segments', key: 'segmentCount', sortable: isTableSortable.value },
        { title: 'Storage', key: 'storage', sortable: isTableSortable.value },
        { title: 'Download', key: 'egress', sortable: isTableSortable.value },
    ];

    if (showRegionTag.value) {
        hdrs.push({ title: 'Location', key: 'location', sortable: isTableSortable.value });
    }

    if (versioningUIEnabled.value) {
        hdrs.push({ title: 'Versioning', key: 'versioning', sortable: isTableSortable.value });
    }

    if (objectLockUIEnabled.value) {
        hdrs.push({ title: 'Lock', key: 'objectLockEnabled', sortable: isTableSortable.value });
    }

    hdrs.push(
        { title: 'Date Created', key: 'since', sortable: isTableSortable.value },
        { title: '', key: 'actions', width: '0', sortable: false },
    );

    return hdrs;
});

/**
 * Returns buckets cursor from store.
 */
const cursor = computed((): BucketCursor => {
    return bucketsStore.state.cursor;
});

/**
 * Returns buckets page from store.
 */
const page = computed((): BucketPage => {
    return bucketsStore.state.page;
});

/**
 * Returns condition if user has to be prompt for passphrase from store.
 */
const promptForPassphrase = computed((): boolean => {
    return bucketsStore.state.promptForPassphrase;
});

/**
 * Returns selected bucket's name from store.
 */
const selectedBucketName = computed((): string => {
    return bucketsStore.state.fileComponentBucketName;
});

/**
 * Returns edge credentials from store.
 */
const edgeCredentials = computed((): EdgeCredentials => {
    return bucketsStore.state.edgeCredentials;
});

/**
 * Returns buckets being deleted from store.
 */
const bucketsBeingDeleted = computed((): Set<string> => bucketsStore.state.bucketsBeingDeleted);

function showLockActionItem(bucket: Bucket): boolean {
    return objectLockUIEnabled.value && bucket.versioning === Versioning.Enabled;
}

/**
 * Fetches bucket using api.
 */
async function fetchBuckets(page = FIRST_PAGE, limit = DEFAULT_PAGE_LIMIT): Promise<void> {
    try {
        await bucketsStore.getBuckets(page, projectsStore.state.selectedProject.id, limit);
        if (areBucketsFetching.value) areBucketsFetching.value = false;
    } catch (error) {
        notify.notifyError(error, AnalyticsErrorEventSource.BUCKET_TABLE);
    }
}

/**
 * Sorts items by provided sort options.
 * We use this to correctly sort columns by value of correct type.
 * @param items
 * @param sortOptions
 */
function sort(items: Bucket[], sortOptions: SortItem[] | undefined): void {
    if (!(sortOptions && sortOptions.length)) {
        items.sort((a, b) => a.name.localeCompare(b.name));
        return;
    }

    const option = sortOptions[0];

    switch (option.key) {
    case 'egress':
        items.sort((a, b) => option.order === 'asc' ? a.egress - b.egress : b.egress - a.egress);
        break;
    case 'storage':
        items.sort((a, b) => option.order === 'asc' ? a.storage - b.storage : b.storage - a.storage);
        break;
    case 'objectCount':
        items.sort((a, b) => option.order === 'asc' ? a.objectCount - b.objectCount : b.objectCount - a.objectCount);
        break;
    case 'segmentCount':
        items.sort((a, b) => option.order === 'asc' ? a.segmentCount - b.segmentCount : b.segmentCount - a.segmentCount);
        break;
    case 'location':
        items.sort((a, b) => option.order === 'asc' ? a.location.localeCompare(b.location) : b.location.localeCompare(a.location));
        break;
    case 'versioning':
        items.sort((a, b) => option.order === 'asc' ? a.versioning.localeCompare(b.versioning) : b.versioning.localeCompare(a.versioning));
        break;
    case 'since':
        items.sort((a, b) => option.order === 'asc' ? a.since.getTime() - b.since.getTime() : b.since.getTime() - a.since.getTime());
        break;
    default:
        items.sort((a, b) => option.order === 'asc' ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name));
    }
}

/**
 * Toggles versioning for the bucket between Suspended and Enabled.
 */
async function onToggleVersioning(bucket: Bucket) {
    withTrialCheck(() => { withManagedPassphraseCheck(() => {
        bucketToToggleVersioning.value = new BucketMetadata(bucket.name, bucket.versioning);
    });});
}

/**
 * Returns helper info based on versioning status.
 */
function getVersioningInfo(status: Versioning): string {
    switch (status) {
    case Versioning.Enabled:
        return 'Version history saved for all objects.';
    case Versioning.Suspended:
        return 'Versioning is currently suspended.';
    case Versioning.NotSupported:
        return 'Versioning is not supported for this bucket.';
    case Versioning.Unversioned:
        return 'This bucket does not have versioning enabled.';
    default:
        return 'Unknown versioning status.';
    }
}

/**
 * Returns helper info based on object lock status.
 */
function getObjectLockInfo(bucket: Bucket): string {
    switch (true) {
    case !bucket.objectLockEnabled:
        return 'Object lock not enabled.';
    case bucket.defaultRetentionMode === NO_MODE_SET:
        return 'Default Mode: None';
    case bucket.defaultRetentionDays !== null:
        return `Default Mode: ${capitalizedMode(bucket.defaultRetentionMode)} / ${bucket.defaultRetentionDays} day${ bucket.defaultRetentionDays > 1 ? 's' : '' } retention`;
    case bucket.defaultRetentionYears !== null:
        return `Default Mode: ${capitalizedMode(bucket.defaultRetentionMode)} / ${bucket.defaultRetentionYears} year${ bucket.defaultRetentionYears > 1 ? 's' : '' } retention`;
    default:
        return 'Unknown object lock status.';
    }
}

/**
 * Returns icon based on versioning status.
 */
function getVersioningIcon(status: Versioning): FunctionalComponent {
    switch (status) {
    case Versioning.Enabled:
        return CircleCheck;
    case Versioning.Suspended:
        return CirclePause;
    case Versioning.NotSupported:
        return CircleX;
    case Versioning.Unversioned:
        return CircleMinus;
    default:
        return CircleHelp;
    }
}

/**
 * Returns chip color based on versioning status.
 */
function getVersioningChipColor(status: Versioning): string {
    switch (status) {
    case Versioning.Enabled:
        return 'success';
    case Versioning.Suspended:
        return 'warning';
    default:
        return 'default';
    }
}

/**
 * Handles update table rows limit event.
 */
function onUpdateLimit(limit: number): void {
    fetchBuckets(FIRST_PAGE, limit);
}

/**
 * Handles update table page event.
 */
function onUpdatePage(page: number): void {
    fetchBuckets(page, cursor.value.limit);
}

/**
 * Handles update table sorting event.
 */
function onUpdateSort(value: SortItem[]): void {
    sortBy.value = value;
}

/**
 * Navigates to bucket page.
 */
function openBucket(bucketName: string): void {
    withManagedPassphraseCheck(async () => {
        if (!bucketName) {
            return;
        }
        bucketsStore.setFileComponentBucketName(bucketName);
        if (!promptForPassphrase.value) {
            if (!edgeCredentials.value.accessKeyId) {
                try {
                    await bucketsStore.setS3Client(projectsStore.state.selectedProject.id);
                } catch (error) {
                    notify.notifyError(error, AnalyticsErrorEventSource.BUCKET_TABLE);
                    return;
                }
            }

            const objCount = bucketsStore.state.page.buckets?.find((bucket) => bucket.name === bucketName)?.objectCount ?? 0;
            obStore.setObjectCountOfSelectedBucket(objCount);

            await router.push({
                name: ROUTES.Bucket.name,
                params: {
                    browserPath: bucketsStore.state.fileComponentBucketName,
                    id: projectsStore.state.selectedProject.urlId,
                },
            });
            return;
        }
        passphraseDialogCallback = () => openBucket(selectedBucketName.value);
        isBucketPassphraseDialogOpen.value = true;
    });
}

/**
 * Handles download bucket action.
 */
function onDownloadBucket(bucketName: string): void {
    withTrialCheck(() => { withManagedPassphraseCheck(async () => {
        if (!bucketName) {
            return;
        }

        function setBucketDownload(): void {
            bucketToDownload.value = bucketName;
            isDownloadPrefixDialogShown.value = true;
        }

        if (promptForPassphrase.value) {
            passphraseDialogCallback = setBucketDownload;

            bucketsStore.setFileComponentBucketName(bucketName);
            isBucketPassphraseDialogOpen.value = true;
            return;
        }

        setBucketDownload();
    });});
}

/**
 * Displays the Bucket Details dialog.
 */
function showBucketDetailsModal(bucketName: string): void {
    bucketDetailsName.value = bucketName;
    isBucketDetailsDialogShown.value = true;
}

/**
 * Displays the Delete Bucket dialog.
 */
function showDeleteBucketDialog(bucketName: string): void {
    bucketToDelete.value = bucketName;
    isDeleteBucketDialogShown.value = true;
}

function showSetBucketObjectLockDialog(bucketName: string): void {
    withTrialCheck(() => { withManagedPassphraseCheck(() => {
        bucketObjectLockName.value = bucketName;
        isSetBucketObjectLockDialogShown.value = true;
    });});
}

/**
 * Displays the Share Bucket dialog.
 */
function showShareBucketDialog(bucketName: string): void {
    withTrialCheck(() => { withManagedPassphraseCheck(() => {
        shareBucketName.value = bucketName;
        if (promptForPassphrase.value) {
            bucketsStore.setFileComponentBucketName(bucketName);
            isBucketPassphraseDialogOpen.value = true;
            passphraseDialogCallback = () => isShareBucketDialogShown.value = true;
            return;
        }
        isShareBucketDialogShown.value = true;
    });});
}

/**
 * Handles page width change.
 */
function resizeHandler(): void {
    pageWidth.value = document.body.clientWidth;
}

/**
 * Handles update table search.
 */
watch(() => search.value, () => {
    clearTimeout(searchTimer.value);

    searchTimer.value = setTimeout(() => {
        bucketsStore.setBucketsSearch(search.value || '');
        fetchBuckets();
    }, 500); // 500ms delay for every new call.
});

watch(() => page.value.totalCount, () => {
    sortBy.value = [{ key: 'name', order: 'asc' }];
});

onMounted(() => {
    window.addEventListener('resize', resizeHandler);

    fetchBuckets();
});

onBeforeUnmount(() => {
    window.removeEventListener('resize', resizeHandler);
    bucketsStore.setBucketsSearch('');
});
</script>

<style scoped lang="scss">
.max-width {
    max-width: 250px;

    @media screen and (width <= 780px) {
        max-width: 400px;
    }

    @media screen and (width <= 620px) {
        max-width: 300px;
    }

    @media screen and (width <= 490px) {
        max-width: 200px;
    }

    @media screen and (width <= 385px) {
        max-width: 100px;
    }
}
</style>
