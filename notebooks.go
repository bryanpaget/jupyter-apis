package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	kubeflowv1 "github.com/StatCan/kubeflow-apis/apis/kubeflow/v1"
	"github.com/andanhm/go-prettytime"
	"github.com/gorilla/mux"
	"gopkg.in/inf.v0"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// DefaultServiceAccountName String.
const DefaultServiceAccountName string = "default-editor"

// SharedMemoryVolumeName String.
const SharedMemoryVolumeName string = "dshm"

// SharedMemoryVolumePath String.
const SharedMemoryVolumePath string = "/dev/shm"

// EnvKfLanguage String.
const EnvKfLanguage string = "KF_LANG"

// StoppedAnnotation is the annotation name present on stopped resources.
const StoppedAnnotation string = "kubeflow-resource-stopped"

// ServerTypeAnnotation is the annotation name representing the server type of the notebook.
const ServerTypeAnnotation string = "notebooks.kubeflow.org/server-type"

// Begin structs necessary for handling volumes
type volrequest struct {
	Mount          string         `json:"mount,omitempty"`
	ExistingSource ExistingSource `json:"existingSource,omitempty"`
	NewPvc         NewPvc         `json:"newPvc,omitempty"`
}

type PersistentVolumeClaim struct {
	ReadOnly  bool    `json:"readOnly"`
	ClaimName *string `json:"claimName"` //https://stackoverflow.com/a/31505089
}
type ExistingSource struct {
	PersistentVolumeClaim PersistentVolumeClaim `json:"persistentVolumeClaim"`
}

type NewPvcMetadata struct {
	Name *string `json:"name"` //https://stackoverflow.com/a/31505089
}
type Requests struct {
	Storage resource.Quantity `json:"storage"`
}
type Resources struct {
	Requests Requests `json:"requests"`
}
type NewPvcSpec struct {
	AccessModes      []corev1.PersistentVolumeAccessMode `json:"accessModes"`
	Resources        Resources                           `json:"resources"`
	StorageClassName string                              `json:"storageClassName,omitempty"`
}
type NewPvc struct {
	NewPvcMetadata NewPvcMetadata `json:"metadata"`
	NewPvcSpec     NewPvcSpec     `json:"spec"`
}

type gpurequest struct {
	Quantity string `json:"num"`
	Vendor   string `json:"vendor"`
}

type newnotebookrequest struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Image              string            `json:"image"`
	CustomImage        string            `json:"customImage"`
	CustomImageCheck   bool              `json:"customImageCheck"`
	CPU                resource.Quantity `json:"cpu"`
	CPULimit           resource.Quantity `json:"cpuLimit"`
	Memory             resource.Quantity `json:"memory"`
	MemoryLimit        resource.Quantity `json:"memoryLimit"`
	GPUs               gpurequest        `json:"gpus"`
	NoWorkspace        bool              `json:"noWorkspace"`
	Workspace          volrequest        `json:"workspace"`
	DataVolumes        []volrequest      `json:"datavols"`
	EnableSharedMemory bool              `json:"shm"`
	Configurations     []string          `json:"configurations"`
	Language           string            `json:"language"`
	ImagePullPolicy    string            `json:"imagePullPolicy"`
	ServerType         string            `json:"serverType"`
	AffinityConfig     string            `json:"affinityConfig"`
	TolerationGroup    string            `json:"tolerationGroup"`
}

type gpuresponse struct {
	Count   resource.Quantity `json:"count"`
	Message string            `json:"message"`
}

type notebookresponse struct {
	Age        string            `json:"age"`
	CPU        *inf.Dec          `json:"cpu"`
	GPUs       gpuresponse       `json:"gpu"`
	Image      string            `json:"image"`
	Memory     resource.Quantity `json:"memory"`
	Name       string            `json:"name"`
	ServerType interface{}       `json:"serverType"`
	Namespace  string            `json:"namespace"`
	ShortImage string            `json:"shortImage"`
	Status     status            `json:"status"`
	Volumes    []string          `json:"volumes"`
	Labels     map[string]string `json:"labels"`
}

type notebooksresponse struct {
	APIResponseBase
	Notebooks []notebookresponse `json:"notebooks"`
}

type updatenotebookrequest struct {
	Stopped bool `json:"stopped"`
}

// notebookPhase is the phase of a notebook.
type notebookPhase string

// status represents the status of a notebook.
type status struct {
	Message string        `json:"message"`
	Phase   notebookPhase `json:"phase"`
	State   string        `json:"state"`
	Key     string        `json:"key"`
}

const (
	// NotebookPhaseReady represents the ready phase of a notebook.
	NotebookPhaseReady notebookPhase = "ready"

	// NotebookPhaseWaiting represents the waiting phase of a notebook.
	NotebookPhaseWaiting notebookPhase = "waiting"

	// NotebookPhaseWarning represents the warning phase of a notebook.
	NotebookPhaseWarning notebookPhase = "warning"

	// NotebookPhaseError represents the error phase of a notebook.
	NotebookPhaseError notebookPhase = "error"

	// NotebookPhaseUnitialized represents the uninitialized phase of a notebook.
	NotebookPhaseUnitialized notebookPhase = "uninitialized"

	// NotebookPhaseUnavailable represents the unavailable phase of a notebook.
	NotebookPhaseUnavailable notebookPhase = "unavailable"

	// NotebookPhaseTerminating represents the terminating phase of a notebook.
	NotebookPhaseTerminating notebookPhase = "terminating"

	// NotebookPhaseStopped represents the stopped phase of a notebook.
	NotebookPhaseStopped notebookPhase = "stopped"
)

// Based on: https://github.com/kubeflow/kubeflow/blob/0e91a2b9cd0c3b6687692b1f1f09ac6070cc6c3e/components/crud-web-apps/jupyter/backend/apps/common/status.py#L9
func processStatus(notebook *kubeflowv1.Notebook, events []*corev1.Event) status {
	// Check if the notebook is bing deleting
	if notebook.DeletionTimestamp != nil {
		return status{
			Message: "Deleting this Notebook Server.",
			Phase:   NotebookPhaseTerminating,
			Key:     "notebookDeleting",
		}
	}

	// Check if the notebook is stopped
	if _, ok := notebook.Annotations[StoppedAnnotation]; ok {
		if notebook.Status.ReadyReplicas == 0 {
			return status{
				Message: "No pods are currently running for this Notebook Server.",
				Phase:   NotebookPhaseStopped,
				Key:     "noPodsRunning",
			}
		}

		return status{
			Message: "Notebook Server is stopping.",
			Phase:   NotebookPhaseTerminating,
			Key:     "notebookStopping",
		}
	}

	// Check the status
	state := notebook.Status.ContainerState

	if notebook.Status.ReadyReplicas == 1 {
		return status{
			Message: "Running",
			Phase:   NotebookPhaseReady,
			Key:     "running",
		}
	}

	if state.Waiting != nil {
		return status{
			Message: state.Waiting.Reason,
			Phase:   NotebookPhaseWaiting,
			Key:     "waitingStatus",
		}
	}

	// Check for more detailed errors
	for _, event := range events {
		if event.Type == corev1.EventTypeWarning {
			return status{
				Message: event.Reason,
				Phase:   NotebookPhaseWarning,
				Key:     "errorEvent",
			}
		}
	}

	return status{
		Message: "Scheduling the Pod",
		Phase:   NotebookPhaseWaiting,
		Key:     "schedulingPod",
	}
}

func (s *server) processGPUs(notebook *kubeflowv1.Notebook) gpuresponse {
	response := gpuresponse{}

	vendors := map[corev1.ResourceName]string{}
	for _, vendor := range s.Config.SpawnerFormDefaults.GPUs.Value.Vendors {
		vendors[corev1.ResourceName(vendor.LimitsKey)] = vendor.UIName
	}

	counts := []string{}
	for vendorKey, vendorName := range vendors {
		if quantity, ok := notebook.Spec.Template.Spec.Containers[0].Resources.Requests[vendorKey]; ok {
			response.Count.Add(quantity)
			counts = append(counts, fmt.Sprintf("%s %s", quantity.String(), vendorName))
		}
	}

	response.Message = strings.Join(counts, ", ")

	return response
}

func (s *server) GetNotebooks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	log.Printf("loading notebooks for %q", namespace)

	notebooks, err := s.listers.notebooks.Notebooks(namespace).List(labels.Everything())
	if err != nil {
		s.error(w, r, err)
		return
	}

	sort.Sort(notebooksByName(notebooks))

	resp := &notebooksresponse{
		APIResponseBase: APIResponseBase{
			Success: true,
			Status:  http.StatusOK,
		},
		Notebooks: make([]notebookresponse, 0),
	}

	for _, notebook := range notebooks {
		// Load events
		allevents, err := s.listers.events.Events(notebook.Namespace).List(labels.Everything())
		if err != nil {
			log.Printf("failed to load events for %s/%s: %v", notebook.Namespace, notebook.Name, err)
		}

		// Filter past events
		events := make([]*corev1.Event, 0)
		for _, event := range allevents {
			if event.InvolvedObject.Kind != "Notebook" || event.InvolvedObject.Name != notebook.Name || event.CreationTimestamp.Before(&notebook.CreationTimestamp) {
				continue
			}

			events = append(events, event)
		}
		sort.Sort(eventsByTimestamp(events))

		imageparts := strings.SplitAfter(notebook.Spec.Template.Spec.Containers[0].Image, "/")

		// Process current status + reason
		status := processStatus(notebook, events)

		volumes := []string{}
		for _, vol := range notebook.Spec.Template.Spec.Volumes {
			volumes = append(volumes, vol.Name)
		}

		cpulimit := resource.Zero.AsDec()
		if req, ok := notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]; ok {
			cpulimit = req.AsDec()
		}

		resp.Notebooks = append(resp.Notebooks, notebookresponse{
			Age:        prettytime.Format(notebook.CreationTimestamp.Time),
			Name:       notebook.Name,
			Namespace:  notebook.Namespace,
			Image:      notebook.Spec.Template.Spec.Containers[0].Image,
			ServerType: notebook.Annotations[ServerTypeAnnotation],
			ShortImage: imageparts[len(imageparts)-1],
			CPU:        cpulimit,
			GPUs:       s.processGPUs(notebook),
			Memory:     notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory],
			Status:     status,
			Volumes:    volumes,
			Labels:     notebook.Labels,
		})
	}

	s.respond(w, r, resp)
}

func (s *server) handleVolume(ctx context.Context, req volrequest, notebook *kubeflowv1.Notebook) error {
	var pvc = corev1.PersistentVolumeClaim{}
	var pvcClaimName string = ""
	// Check if it is a new PVC by checking if the value exists https://stackoverflow.com/a/31505089
	if req.NewPvc.NewPvcMetadata.Name != nil {
		pvcClaimName = *req.NewPvc.NewPvcMetadata.Name
		if _, ok := notebook.GetObjectMeta().GetLabels()["notebook.statcan.gc.ca/protected-b"]; ok {
			pvc = corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      *req.NewPvc.NewPvcMetadata.Name,
					Namespace: notebook.Namespace,
					Labels:    map[string]string{"data.statcan.gc.ca/classification": "protected-b"},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: req.NewPvc.NewPvcSpec.AccessModes,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: req.NewPvc.NewPvcSpec.Resources.Requests.Storage,
						},
					},
				},
			}
		} else {
			// Create the PVC
			pvc = corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      *req.NewPvc.NewPvcMetadata.Name,
					Namespace: notebook.Namespace,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: req.NewPvc.NewPvcSpec.AccessModes,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: req.NewPvc.NewPvcSpec.Resources.Requests.Storage,
						},
					},
				},
			}
		}
		// Add the storage class, if set and not set to an "empty" value
		if req.NewPvc.NewPvcSpec.StorageClassName != "" &&
			req.NewPvc.NewPvcSpec.StorageClassName != "{none}" &&
			req.NewPvc.NewPvcSpec.StorageClassName != "{empty}" {
			pvc.Spec.StorageClassName = &req.NewPvc.NewPvcSpec.StorageClassName
		}

		if _, err := s.clientsets.kubernetes.CoreV1().PersistentVolumeClaims(notebook.Namespace).Create(ctx, &pvc, metav1.CreateOptions{}); err != nil {
			return err
		}
		// Add the volume and volume mount to the notebook spec
		notebook.Spec.Template.Spec.Volumes = append(notebook.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: pvcClaimName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcClaimName,
				},
			},
		})

		notebook.Spec.Template.Spec.Containers[0].VolumeMounts = append(notebook.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      pvcClaimName,
			MountPath: req.Mount,
		})

	} else if req.ExistingSource.PersistentVolumeClaim.ClaimName != nil {
		pvcClaimName = *req.ExistingSource.PersistentVolumeClaim.ClaimName
		// Add the volume and volume mount to the notebook spec
		notebook.Spec.Template.Spec.Volumes = append(notebook.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: pvcClaimName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcClaimName,
				},
			},
		})

		notebook.Spec.Template.Spec.Containers[0].VolumeMounts = append(notebook.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      pvcClaimName,
			MountPath: req.Mount,
		})
	}

	return nil
}

func (s *server) NewNotebook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	// Read the incoming notebook
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.error(w, r, err)
		return
	}
	defer r.Body.Close()

	var req newnotebookrequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		s.error(w, r, err)
		return
	}

	image := req.Image
	if req.CustomImageCheck {
		image = req.CustomImage
	}
	if s.Config.SpawnerFormDefaults.Image.ReadOnly {
		image = s.Config.SpawnerFormDefaults.Image.Value
	}

	// Setup the notebook
	notebook := kubeflowv1.Notebook{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
			Labels:    make(map[string]string),
			Annotations: map[string]string{
				ServerTypeAnnotation: req.ServerType,
			},
		},
		Spec: kubeflowv1.NotebookSpec{
			Template: kubeflowv1.NotebookTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: DefaultServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  req.Name,
							Image: image,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{},
								Limits:   corev1.ResourceList{},
							},
						},
					},
				},
			},
		},
	}

	// Resources
	if s.Config.SpawnerFormDefaults.CPU.ReadOnly {
		val, err := resource.ParseQuantity(s.Config.SpawnerFormDefaults.CPU.Value)
		if err != nil {
			s.error(w, r, err)
			return
		}

		notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = val
		notebook.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = val
	} else {
		notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = req.CPU
		notebook.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = req.CPULimit
	}

	if s.Config.SpawnerFormDefaults.Memory.ReadOnly {
		val, err := resource.ParseQuantity(s.Config.SpawnerFormDefaults.Memory.Value)
		if err != nil {
			s.error(w, r, err)
			return
		}

		notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = val
		notebook.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = val
	} else {
		notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = req.Memory
		notebook.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = req.MemoryLimit
	}

	// Add configuration items
	if s.Config.SpawnerFormDefaults.Configurations.ReadOnly {
		for _, config := range s.Config.SpawnerFormDefaults.Configurations.Value {
			notebook.ObjectMeta.Labels[config] = "true"
		}
	} else {
		for _, config := range req.Configurations {
			notebook.ObjectMeta.Labels[config] = "true"
		}
	}

	// Add workspace volume
	if s.Config.SpawnerFormDefaults.WorkspaceVolume.ReadOnly { //only gets hit on readonly in spawner config, don't have this on often.
		size, err := resource.ParseQuantity(s.Config.SpawnerFormDefaults.WorkspaceVolume.Value.NewPvc.Spec.Resources.Requests.Storage)
		if err != nil {
			s.error(w, r, err)
			return
		}

		workspaceVol := &volrequest{
			Mount: s.Config.SpawnerFormDefaults.WorkspaceVolume.Value.Mount,
			NewPvc: NewPvc{
				NewPvcMetadata: NewPvcMetadata{
					Name: &s.Config.SpawnerFormDefaults.WorkspaceVolume.Value.NewPvc.Metadata.Name,
				},
				NewPvcSpec: NewPvcSpec{
					Resources: Resources{
						Requests: Requests{
							Storage: size,
						},
					},
					AccessModes:      s.Config.SpawnerFormDefaults.WorkspaceVolume.Value.NewPvc.Spec.AccessModes,
					StorageClassName: "{none}",
				},
			},
		}
		err = s.handleVolume(r.Context(), *workspaceVol, &notebook)
		if err != nil {
			s.error(w, r, err)
			return
		}

	} else if !req.NoWorkspace {
		req.Workspace.Mount = s.Config.SpawnerFormDefaults.WorkspaceVolume.Value.Mount
		err = s.handleVolume(r.Context(), req.Workspace, &notebook)
		if err != nil {
			s.error(w, r, err)
			return
		}
	}

	if s.Config.SpawnerFormDefaults.DataVolumes.ReadOnly {
		for _, volreq := range s.Config.SpawnerFormDefaults.DataVolumes.Value {
			size, err := resource.ParseQuantity(s.Config.SpawnerFormDefaults.WorkspaceVolume.Value.NewPvc.Spec.Resources.Requests.Storage)
			if err != nil {
				s.error(w, r, err)
				return
			}
			vol := volrequest{
				Mount: volreq.Value.Mount,
				NewPvc: NewPvc{
					NewPvcMetadata: NewPvcMetadata{
						Name: &volreq.Value.NewPvc.Metadata.Name,
					},
					NewPvcSpec: NewPvcSpec{
						Resources: Resources{
							Requests: Requests{
								Storage: size,
							},
						},
						AccessModes:      req.Workspace.NewPvc.NewPvcSpec.AccessModes,
						StorageClassName: "{none}",
					},
				},
			}
			err = s.handleVolume(r.Context(), vol, &notebook)
			if err != nil {
				s.error(w, r, err)
				return
			}

		}
	} else {
		for _, volreq := range req.DataVolumes {
			err = s.handleVolume(r.Context(), volreq, &notebook)
			if err != nil {
				s.error(w, r, err)
				return
			}
		}
	}

	// Add shared memory, if enabled
	if (s.Config.SpawnerFormDefaults.Shm.ReadOnly && s.Config.SpawnerFormDefaults.Shm.Value) || (!s.Config.SpawnerFormDefaults.Shm.ReadOnly && req.EnableSharedMemory) {
		notebook.Spec.Template.Spec.Volumes = append(notebook.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: SharedMemoryVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})

		notebook.Spec.Template.Spec.Containers[0].VolumeMounts = append(notebook.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      SharedMemoryVolumeName,
			MountPath: SharedMemoryVolumePath,
		})
	}

	// Add GPU
	if s.Config.SpawnerFormDefaults.GPUs.ReadOnly {
		if s.Config.SpawnerFormDefaults.GPUs.Value.Num != "none" {
			qty, err := resource.ParseQuantity(s.Config.SpawnerFormDefaults.GPUs.Value.Num)
			if err != nil {
				s.error(w, r, err)
				return
			}

			notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceName(s.Config.SpawnerFormDefaults.GPUs.Value.Vendor)] = qty
			notebook.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceName(s.Config.SpawnerFormDefaults.GPUs.Value.Vendor)] = qty
		}
	} else {
		if req.GPUs.Quantity != "none" {
			qty, err := resource.ParseQuantity(req.GPUs.Quantity)
			if err != nil {
				s.error(w, r, err)
				return
			}

			notebook.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceName(req.GPUs.Vendor)] = qty
			notebook.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceName(req.GPUs.Vendor)] = qty
		}
	}

	// Add tolerations
	if req.TolerationGroup != "none" {
		for _, tolerationGroup := range s.Config.SpawnerFormDefaults.TolerationGroup.Options {
			if tolerationGroup.GroupKey != req.TolerationGroup {
				continue
			}

			notebook.Spec.Template.Spec.Tolerations = tolerationGroup.Tolerations
		}
	}

	// Add affinity
	if req.AffinityConfig != "none" {
		for _, affinityConfig := range s.Config.SpawnerFormDefaults.AffinityConfig.Options {
			if affinityConfig.ConfigKey != req.AffinityConfig {
				continue
			}

			notebook.Spec.Template.Spec.Affinity = &affinityConfig.Affinity
		}
	}

	//Add Language
	//Validate that the language format is valid (language[_territory])
	if req.Language != "" {
		match, err := regexp.MatchString("^[[:alpha:]]{2}(_[[:alpha:]]{2})?$", req.Language)
		if err != nil || !match {
			var errLanguageFormat = errors.New("Error: the value of KF_LANG environment variable ('" + req.Language + "') is not a valid format (e.g 'en', 'en_US', ...)")
			s.error(w, r, errLanguageFormat)
			return
		}
		notebook.Spec.Template.Spec.Containers[0].Env = append(notebook.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  EnvKfLanguage,
			Value: req.Language,
		})
	}
	// Add empty dir on protected-b notebooks
	if _, ok := notebook.GetObjectMeta().GetLabels()["notebook.statcan.gc.ca/protected-b"]; ok {

		notebook.Spec.Template.Spec.Volumes = append(notebook.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "protb-nb",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})
		notebook.Spec.Template.Spec.Containers[0].VolumeMounts = append(notebook.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "protb-nb",
			MountPath: "/etc/protb",
		})
	}

	// Add imagePullPolicy
	if req.ImagePullPolicy == "Always" || req.ImagePullPolicy == "Never" || req.ImagePullPolicy == "IfNotPresent" {
		notebook.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullPolicy(req.ImagePullPolicy)
	}
	log.Printf("creating notebook %q for %q", notebook.ObjectMeta.Name, namespace)

	// Submit the notebook to the API server
	_, err = s.clientsets.kubeflow.KubeflowV1().Notebooks(namespace).Create(r.Context(), &notebook, metav1.CreateOptions{})
	if err != nil {
		s.error(w, r, err)
		return
	}

	s.respond(w, r, &APIResponseBase{
		Success: true,
		Status:  http.StatusOK,
	})
}

func (s *server) DeleteNotebook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	notebook := vars["notebook"]

	log.Printf("deleting notebook %q for %q", notebook, namespace)

	propagation := metav1.DeletePropagationForeground
	err := s.clientsets.kubeflow.KubeflowV1().Notebooks(namespace).Delete(r.Context(), notebook, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		s.error(w, r, err)
		return
	}

	s.respond(w, r, &APIResponseBase{
		Success: true,
		Status:  http.StatusOK,
	})
}

func (s *server) UpdateNotebook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespaceName := vars["namespace"]
	notebookName := vars["notebook"]

	log.Printf("deleting notebook %q for %q", notebookName, namespaceName)

	// Read the incoming notebook
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.error(w, r, err)
		return
	}
	defer r.Body.Close()

	var req updatenotebookrequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		s.error(w, r, err)
		return
	}

	// Read existing notebook
	notebook, err := s.listers.notebooks.Notebooks(namespaceName).Get(notebookName)
	if err != nil {
		s.error(w, r, err)
		return
	}

	update := false
	updatedNotebook := notebook.DeepCopy()

	// Compare start/stopped state
	if _, ok := notebook.Annotations[StoppedAnnotation]; ok != req.Stopped {
		update = true

		if req.Stopped {
			// Set the stopped annotation
			if updatedNotebook.Annotations == nil {
				updatedNotebook.Annotations = map[string]string{}
			}

			updatedNotebook.Annotations[StoppedAnnotation] = time.Now().Format(time.RFC3339)
		} else {
			// Remove the stopped annotation
			delete(updatedNotebook.Annotations, StoppedAnnotation)
		}
	}

	if update {
		_, err = s.clientsets.kubeflow.KubeflowV1().Notebooks(namespaceName).Update(r.Context(), updatedNotebook, metav1.UpdateOptions{})
		if err != nil {
			s.error(w, r, err)
			return
		}
	}

	s.respond(w, r, &APIResponseBase{
		Success: true,
		Status:  http.StatusOK,
	})
}
