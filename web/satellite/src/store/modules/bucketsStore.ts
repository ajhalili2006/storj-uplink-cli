// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

import { reactive } from 'vue';
import { defineStore } from 'pinia';
import S3 from 'aws-sdk/clients/s3';

import { Bucket, BucketCursor, BucketPage, BucketsApi } from '@/types/buckets';
import { BucketsApiGql } from '@/api/buckets';
import { AccessGrant, EdgeCredentials } from '@/types/accessGrants';
import { useAccessGrantsStore } from '@/store/modules/accessGrantsStore';
import { useProjectsStore } from '@/store/modules/projectsStore';
import { MetaUtils } from '@/utils/meta';
import { useAppStore } from '@/store/modules/appStore';
import { MODALS } from '@/utils/constants/appStatePopUps';

const BUCKETS_PAGE_LIMIT = 7;
const FIRST_PAGE = 1;
export const FILE_BROWSER_AG_NAME = 'Web file browser API key';

export class BucketsState {
    public allBucketNames: string[] = [];
    public cursor: BucketCursor = { limit: BUCKETS_PAGE_LIMIT, search: '', page: FIRST_PAGE };
    public page: BucketPage = { buckets: new Array<Bucket>(), currentPage: 1, pageCount: 1, offset: 0, limit: BUCKETS_PAGE_LIMIT, search: '', totalCount: 0 };
    public edgeCredentials: EdgeCredentials = new EdgeCredentials();
    public edgeCredentialsForDelete: EdgeCredentials = new EdgeCredentials();
    public edgeCredentialsForCreate: EdgeCredentials = new EdgeCredentials();
    public s3Client: S3 = new S3({
        s3ForcePathStyle: true,
        signatureVersion: 'v4',
        httpOptions: { timeout: 0 },
    });
    public s3ClientForDelete: S3 = new S3({
        s3ForcePathStyle: true,
        signatureVersion: 'v4',
        httpOptions: { timeout: 0 },
    });
    public s3ClientForCreate: S3 = new S3({
        s3ForcePathStyle: true,
        signatureVersion: 'v4',
        httpOptions: { timeout: 0 },
    });
    public apiKey = '';
    public passphrase = '';
    public promptForPassphrase = true;
    public fileComponentBucketName = '';
    public leaveRoute = '';
}

export const useBucketsStore = defineStore('buckets', () => {
    const state = reactive<BucketsState>(new BucketsState());

    const api: BucketsApi = new BucketsApiGql();

    function setBucketsSearch(search: string): void {
        state.cursor.search = search;
    }

    function clearBucketsState(): void {
        state.allBucketNames = [];
        state.cursor = new BucketCursor('', BUCKETS_PAGE_LIMIT, FIRST_PAGE);
        state.page = new BucketPage([], '', BUCKETS_PAGE_LIMIT, 0, 1, 1, 0);
    }

    async function fetchBuckets(projectID: string, page: number): Promise<void> {
        const before = new Date();
        state.cursor.page = page;

        state.page = await api.get(projectID, before, state.cursor);
    }

    async function fetchAllBucketsNames(projectID: string): Promise<void> {
        state.allBucketNames = await api.getAllBucketNames(projectID);
    }

    function setPromptForPassphrase(value: boolean): void {
        state.promptForPassphrase = value;
    }

    function setApiKey(apiKey: string): void {
        state.apiKey = apiKey;
    }

    function setEdgeCredentials(credentials: EdgeCredentials): void {
        state.edgeCredentials = credentials;
    }

    function setEdgeCredentialsForDelete(credentials: EdgeCredentials): void {
        state.edgeCredentialsForDelete = credentials;

        const s3Config = {
            accessKeyId: state.edgeCredentialsForDelete.accessKeyId,
            secretAccessKey: state.edgeCredentialsForDelete.secretKey,
            endpoint: state.edgeCredentialsForDelete.endpoint,
            s3ForcePathStyle: true,
            signatureVersion: 'v4',
            httpOptions: { timeout: 0 },
        };

        state.s3ClientForDelete = new S3(s3Config);
    }

    function setEdgeCredentialsForCreate(credentials: EdgeCredentials): void {
        state.edgeCredentialsForCreate = credentials;

        const s3Config = {
            accessKeyId: state.edgeCredentialsForCreate.accessKeyId,
            secretAccessKey: state.edgeCredentialsForCreate.secretKey,
            endpoint: state.edgeCredentialsForCreate.endpoint,
            s3ForcePathStyle: true,
            signatureVersion: 'v4',
            httpOptions: { timeout: 0 },
        };

        state.s3ClientForCreate = new S3(s3Config);
    }

    async function setS3Client(projectID: string): Promise<void> {
        const {
            createAccessGrant,
            deleteAccessGrantByNameAndProjectID,
            accessGrantsState,
            getEdgeCredentials,
        } = useAccessGrantsStore();

        if (!state.apiKey) {
            await deleteAccessGrantByNameAndProjectID(projectID, FILE_BROWSER_AG_NAME);
            const cleanAPIKey: AccessGrant = await createAccessGrant(projectID, FILE_BROWSER_AG_NAME);
            setApiKey(cleanAPIKey.secret);
        }

        const now = new Date();
        const inThreeDays = new Date(now.setDate(now.getDate() + 3));

        const worker = accessGrantsState.accessGrantsWebWorker;
        if (!worker) {
            throw new Error('Worker is not defined');
        }

        worker.onerror = (error: ErrorEvent) => {
            throw new Error(error.message);
        };

        await worker.postMessage({
            'type': 'SetPermission',
            'isDownload': true,
            'isUpload': true,
            'isList': true,
            'isDelete': true,
            'notAfter': inThreeDays.toISOString(),
            'buckets': [],
            'apiKey': state.apiKey,
        });

        const grantEvent: MessageEvent = await new Promise(resolve => worker.onmessage = resolve);
        if (grantEvent.data.error) {
            throw new Error(grantEvent.data.error);
        }

        const { getProjectSalt } = useProjectsStore();

        const salt = await getProjectSalt(projectID);
        const satelliteNodeURL: string = MetaUtils.getMetaContent('satellite-nodeurl');

        if (!state.passphrase) {
            throw new Error('Passphrase can\'t be empty');
        }

        worker.postMessage({
            'type': 'GenerateAccess',
            'apiKey': grantEvent.data.value,
            'passphrase': state.passphrase,
            'salt': salt,
            'satelliteNodeURL': satelliteNodeURL,
        });

        const accessGrantEvent: MessageEvent = await new Promise(resolve => worker.onmessage = resolve);
        if (accessGrantEvent.data.error) {
            throw new Error(accessGrantEvent.data.error);
        }

        const accessGrant = accessGrantEvent.data.value;
        state.edgeCredentials = await getEdgeCredentials(accessGrant);

        const s3Config = {
            accessKeyId: state.edgeCredentials.accessKeyId,
            secretAccessKey: state.edgeCredentials.secretKey,
            endpoint: state.edgeCredentials.endpoint,
            s3ForcePathStyle: true,
            signatureVersion: 'v4',
            httpOptions: { timeout: 0 },
        };

        state.s3Client = new S3(s3Config);
    }

    function setPassphrase(passphrase: string): void {
        state.passphrase = passphrase;
    }

    function setFileComponentBucketName(bucketName: string): void {
        state.fileComponentBucketName = bucketName;
    }

    async function createBucket(name: string): Promise<void> {
        await state.s3Client.createBucket({
            Bucket: name,
        }).promise();
    }

    async function createBucketWithNoPassphrase(name: string): Promise<void> {
        await state.s3ClientForCreate.createBucket({
            Bucket: name,
        }).promise();
    }

    async function deleteBucket(name: string): Promise<void> {
        await state.s3ClientForDelete.deleteBucket({
            Bucket: name,
        }).promise();
    }

    async function getObjectsCount(name: string): Promise<number> {
        const response =  await state.s3Client.listObjectsV2({
            Bucket: name,
        }).promise();

        return response.KeyCount === undefined ? 0 : response.KeyCount;
    }

    function clearObjects(): void {
        state.apiKey = '';
        state.passphrase = '';
        state.promptForPassphrase = true;
        state.edgeCredentials = new EdgeCredentials();
        state.edgeCredentialsForDelete = new EdgeCredentials();
        state.edgeCredentialsForCreate = new EdgeCredentials();
        state.s3Client = new S3({
            s3ForcePathStyle: true,
            signatureVersion: 'v4',
            httpOptions: { timeout: 0 },
        });
        state.s3ClientForDelete = new S3({
            s3ForcePathStyle: true,
            signatureVersion: 'v4',
            httpOptions: { timeout: 0 },
        });
        state.s3ClientForCreate = new S3({
            s3ForcePathStyle: true,
            signatureVersion: 'v4',
            httpOptions: { timeout: 0 },
        });
        state.fileComponentBucketName = '';
        state.leaveRoute = '';
    }

    function checkOngoingUploads(uploadingLength: number, leaveRoute: string): boolean {
        if (!uploadingLength) {
            return false;
        }

        state.leaveRoute = leaveRoute;

        const { updateActiveModal } = useAppStore();
        updateActiveModal(MODALS.uploadCancelPopup);

        return true;
    }

    return {
        bucketsState: state,
        setBucketsSearch,
        clearBucketsState,
        fetchBuckets,
        fetchAllBucketsNames,
    };
});
